package crawler

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/service"
)

type Crawler struct {
	logger   *zap.Logger
	pool     *service.WorkerPool
	tracker  *service.ProcessedTracker
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Configuration
	config Config
}

type Config struct {
	// Worker pool configuration
	NumWorkers   int
	JobQueueSize int

	// Crawl intervals
	MissingDataInterval time.Duration
	StaleDataInterval   time.Duration
	PlaylistInterval    time.Duration
	DiscoveryInterval   time.Duration

	// Stale data threshold
	StaleDataThreshold time.Duration

	// Playlist to process weekly
	WeeklyPlaylistID string
}

func New(logger *zap.Logger) *Crawler {
	config := Config{
		NumWorkers:          32,
		JobQueueSize:        100000,
		MissingDataInterval: 1 * time.Hour,
		StaleDataInterval:   24 * time.Hour,
		PlaylistInterval:    7 * 24 * time.Hour, // Weekly
		DiscoveryInterval:   6 * time.Hour,
		StaleDataThreshold:  30 * 24 * time.Hour, // 30 days
		WeeklyPlaylistID:    os.Getenv("CRAWLER_WEEKLY_PLAYLIST_ID"),
	}

	return &Crawler{
		logger:   logger,
		stopChan: make(chan struct{}),
		config:   config,
	}
}

func (c *Crawler) Start() {
	c.pool = service.NewWorkerPool(c.config.NumWorkers, c.config.JobQueueSize)
	c.tracker = service.NewProcessedTracker()

	var jobWg sync.WaitGroup
	c.pool.Start(&jobWg, c.tracker)

	// Start error collection
	c.wg.Add(1)
	go c.collectErrors()

	// Start crawl schedulers
	c.wg.Add(4)
	go c.scheduleMissingDataCrawl(&jobWg)
	go c.scheduleStaleDataCrawl(&jobWg)
	go c.schedulePlaylistCrawl(&jobWg)
	go c.scheduleDiscoveryCrawl(&jobWg)
}

func (c *Crawler) Stop() {
	close(c.stopChan)
	c.pool.Stop()
	c.wg.Wait()
}

func (c *Crawler) collectErrors() {
	defer c.wg.Done()

	for err := range c.pool.GetResultsChan() {
		if err != nil {
			c.logger.Error("Crawler job error", zap.Error(err))
		}
	}
}

func (c *Crawler) scheduleMissingDataCrawl(jobWg *sync.WaitGroup) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.MissingDataInterval)
	defer ticker.Stop()

	// Run immediately on start
	c.crawlMissingData(jobWg)

	for {
		select {
		case <-ticker.C:
			c.crawlMissingData(jobWg)
		case <-c.stopChan:
			return
		}
	}
}

func (c *Crawler) scheduleStaleDataCrawl(jobWg *sync.WaitGroup) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.StaleDataInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.crawlStaleData(jobWg)
		case <-c.stopChan:
			return
		}
	}
}

func (c *Crawler) schedulePlaylistCrawl(jobWg *sync.WaitGroup) {
	defer c.wg.Done()

	if c.config.WeeklyPlaylistID == "" {
		c.logger.Warn("Weekly playlist ID not configured, skipping playlist crawl")
		return
	}

	ticker := time.NewTicker(c.config.PlaylistInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.crawlPlaylist(jobWg)
		case <-c.stopChan:
			return
		}
	}
}

func (c *Crawler) scheduleDiscoveryCrawl(jobWg *sync.WaitGroup) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.DiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.crawlMissingEntities(jobWg)
		case <-c.stopChan:
			return
		}
	}
}

func (c *Crawler) crawlMissingData(jobWg *sync.WaitGroup) {
	c.logger.Info("Starting missing data crawl")
	startTime := time.Now()

	// Create stage context
	stageWg := &sync.WaitGroup{}
	stageCtx := &service.StageContext{
		Wg:   stageWg,
		Name: "missing_data_crawl",
	}

	// Queue job
	job := &RefetchMissingDataJob{
		logger: c.logger,
	}

	jobWg.Add(1)
	stageWg.Add(1)
	c.pool.SubmitWithStage(job, jobWg, stageCtx)

	// Wait for this stage to complete
	stageWg.Wait()

	c.logger.Info("Completed missing data crawl",
		zap.Duration("duration", time.Since(startTime)))
}

func (c *Crawler) crawlStaleData(jobWg *sync.WaitGroup) {
	c.logger.Info("Starting stale data crawl")
	startTime := time.Now()

	// Create stage context
	stageWg := &sync.WaitGroup{}
	stageCtx := &service.StageContext{
		Wg:   stageWg,
		Name: "stale_data_crawl",
	}

	// Queue job
	job := &RefreshStaleDataJob{
		logger:    c.logger,
		threshold: c.config.StaleDataThreshold,
	}

	jobWg.Add(1)
	stageWg.Add(1)
	c.pool.SubmitWithStage(job, jobWg, stageCtx)

	// Wait for this stage to complete
	stageWg.Wait()

	c.logger.Info("Completed stale data crawl",
		zap.Duration("duration", time.Since(startTime)))
}

func (c *Crawler) crawlPlaylist(jobWg *sync.WaitGroup) {
	c.logger.Info("Starting playlist crawl",
		zap.String("playlistId", c.config.WeeklyPlaylistID))
	startTime := time.Now()

	// Create stage context
	stageWg := &sync.WaitGroup{}
	stageCtx := &service.StageContext{
		Wg:   stageWg,
		Name: "playlist_crawl",
	}

	// Queue job
	job := &ProcessPlaylistJob{
		logger:     c.logger,
		playlistID: c.config.WeeklyPlaylistID,
		deep:       true, // Process all artist data
	}

	jobWg.Add(1)
	stageWg.Add(1)
	c.pool.SubmitWithStage(job, jobWg, stageCtx)

	// Wait for this stage to complete
	stageWg.Wait()

	c.logger.Info("Completed playlist crawl",
		zap.String("playlistId", c.config.WeeklyPlaylistID),
		zap.Duration("duration", time.Since(startTime)))
}

func (c *Crawler) crawlMissingEntities(jobWg *sync.WaitGroup) {
	c.logger.Info("Starting missing entities crawl")
	startTime := time.Now()

	// Create stage context
	stageWg := &sync.WaitGroup{}
	stageCtx := &service.StageContext{
		Wg:   stageWg,
		Name: "missing_entities_crawl",
	}

	// Queue job
	job := &DiscoverMissingEntitiesJob{
		logger: c.logger,
	}

	jobWg.Add(1)
	stageWg.Add(1)
	c.pool.SubmitWithStage(job, jobWg, stageCtx)

	// Wait for this stage to complete
	stageWg.Wait()

	c.logger.Info("Completed missing entities crawl",
		zap.Duration("duration", time.Since(startTime)))
}
