package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock for pgxpool.Pool
type MockPool struct {
	mock.Mock
}

func (m *MockPool) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *MockPool) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	args := m.Called(append([]interface{}{ctx, sql}, arguments...)...)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *MockPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	mockArgs := m.Called(append([]interface{}{ctx, sql}, args...)...)
	return mockArgs.Get(0).(pgx.Rows), mockArgs.Error(1)
}

func (m *MockPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	mockArgs := m.Called(append([]interface{}{ctx, sql}, args...)...)
	return mockArgs.Get(0).(pgx.Row)
}

func (m *MockPool) Close() {
	m.Called()
}

// Mock for pgx.Tx
type MockTx struct {
	mock.Mock
}

func (m *MockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *MockTx) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTx) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	args := m.Called(ctx, tableName, columnNames, rowSrc)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	args := m.Called(ctx, b)
	return args.Get(0).(pgx.BatchResults)
}

func (m *MockTx) LargeObjects() pgx.LargeObjects {
	args := m.Called()
	return args.Get(0).(pgx.LargeObjects)
}

func (m *MockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	args := m.Called(ctx, name, sql)
	return args.Get(0).(*pgconn.StatementDescription), args.Error(1)
}

func (m *MockTx) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	args := m.Called(append([]interface{}{ctx, sql}, arguments...)...)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *MockTx) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	mockArgs := m.Called(append([]interface{}{ctx, sql}, args...)...)
	return mockArgs.Get(0).(pgx.Rows), mockArgs.Error(1)
}

func (m *MockTx) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	mockArgs := m.Called(append([]interface{}{ctx, sql}, args...)...)
	return mockArgs.Get(0).(pgx.Row)
}

// Mock for pgx.BatchResults
type MockBatchResults struct {
	mock.Mock
}

func (m *MockBatchResults) Exec() (pgconn.CommandTag, error) {
	args := m.Called()
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *MockBatchResults) Query() (pgx.Rows, error) {
	args := m.Called()
	return args.Get(0).(pgx.Rows), args.Error(1)
}

func (m *MockBatchResults) QueryRow() pgx.Row {
	args := m.Called()
	return args.Get(0).(pgx.Row)
}

func (m *MockBatchResults) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Original functions to be replaced with mocks
var (
	originalGetDB = getDB
)

// Setup and teardown helpers
func setupDBTest() (*MockPool, *MockTx, *MockBatchResults) {
	mockPool := new(MockPool)
	mockTx := new(MockTx)
	mockBatchResults := new(MockBatchResults)

	// Replace the getDB function with a mock
	getDB = func() (*pgxpool.Pool, error) {
		return nil, nil // This will be overridden in each test
	}

	return mockPool, mockTx, mockBatchResults
}

func teardownDBTest() {
	// Restore original functions
	getDB = originalGetDB
}

func TestSaveUser(t *testing.T) {
	mockPool, _, _ := setupDBTest()
	defer teardownDBTest()

	// Create a test user
	testUser := &User{
		UserId:      "test-user-id",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Country:     "US",
		Followers:   100,
		Product:     "premium",
		ImageURLs:   []string{"http://example.com/image.jpg"},
	}

	// Mock the getDB function
	getDB = func() (*pgxpool.Pool, error) {
		return mockPool, nil
	}

	// Mock the Exec function
	mockCommandTag := pgconn.CommandTag("INSERT 1")
	mockPool.On("Exec",
		context.Background(),
		InsertUserQuery,
		testUser.UserId,
		testUser.Email,
		testUser.DisplayName,
		testUser.Country,
		testUser.Followers,
		testUser.Product,
		testUser.ImageURLs,
	).Return(mockCommandTag, nil)

	// Call the function under test
	err := SaveUser(testUser)

	// Assert the result
	assert.NoError(t, err)
	mockPool.AssertExpectations(t)
}

func TestSaveUser_DBConnectionError(t *testing.T) {
	_, _, _ := setupDBTest()
	defer teardownDBTest()

	// Create a test user
	testUser := &User{
		UserId:      "test-user-id",
		Email:       "test@example.com",
		DisplayName: "Test User",
	}

	// Mock the getDB function to return an error
	getDB = func() (*pgxpool.Pool, error) {
		return nil, errors.New("connection error")
	}

	// Call the function under test
	err := SaveUser(testUser)

	// Assert the result
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection error")
}

func TestSaveUser_ExecError(t *testing.T) {
	mockPool, _, _ := setupDBTest()
	defer teardownDBTest()

	// Create a test user
	testUser := &User{
		UserId:      "test-user-id",
		Email:       "test@example.com",
		DisplayName: "Test User",
	}

	// Mock the getDB function
	getDB = func() (*pgxpool.Pool, error) {
		return mockPool, nil
	}

	// Mock the Exec function to return an error
	mockPool.On("Exec",
		context.Background(),
		InsertUserQuery,
		testUser.UserId,
		testUser.Email,
		testUser.DisplayName,
		testUser.Country,
		testUser.Followers,
		testUser.Product,
		testUser.ImageURLs,
	).Return(pgconn.CommandTag(""), errors.New("exec error"))

	// Call the function under test
	err := SaveUser(testUser)

	// Assert the result
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error creating user record")
	mockPool.AssertExpectations(t)
}

func TestSaveTracks(t *testing.T) {
	mockPool, mockTx, mockBatchResults := setupDBTest()
	defer teardownDBTest()

	// Create test tracks
	testTracks := []*Track{
		{
			TrackId:    "track1",
			Name:       "Track 1",
			ArtistIds:  []string{"artist1"},
			AlbumId:    "album1",
			Popularity: 80,
			DurationMS: 180000,
			AudioFeatures: &AudioFeatures{
				Tempo: 120.0,
			},
		},
		{
			TrackId:    "track2",
			Name:       "Track 2",
			ArtistIds:  []string{"artist2"},
			AlbumId:    "album2",
			Popularity: 75,
			DurationMS: 210000,
			AudioFeatures: &AudioFeatures{
				Tempo: 130.0,
			},
		},
	}

	// Mock the getDB function
	getDB = func() (*pgxpool.Pool, error) {
		return mockPool, nil
	}

	// Mock the Begin function
	mockPool.On("Begin", context.Background()).Return(mockTx, nil)

	// Mock the SendBatch function
	mockTx.On("SendBatch", context.Background(), mock.Anything).Return(mockBatchResults)

	// Mock the Exec function for batch results
	mockCommandTag := pgconn.CommandTag("INSERT 1")
	mockBatchResults.On("Exec").Return(mockCommandTag, nil).Times(4) // 2 tracks * 2 queries (track + relation)

	// Mock the Close function for batch results
	mockBatchResults.On("Close").Return(nil).Times(2) // 2 batches (tracks + relations)

	// Mock the Commit function
	mockTx.On("Commit", context.Background()).Return(nil)

	// Mock the Rollback function (should not be called in this test)
	mockTx.On("Rollback", context.Background()).Return(nil)

	// Call the function under test
	err := SaveTracks("test-user-id", testTracks, "test-source")

	// Assert the result
	assert.NoError(t, err)
	mockPool.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	mockBatchResults.AssertExpectations(t)
}

func TestSaveTracks_DBConnectionError(t *testing.T) {
	_, _, _ := setupDBTest()
	defer teardownDBTest()

	// Create test tracks
	testTracks := []*Track{
		{
			TrackId: "track1",
			Name:    "Track 1",
		},
	}

	// Mock the getDB function to return an error
	getDB = func() (*pgxpool.Pool, error) {
		return nil, errors.New("connection error")
	}

	// Call the function under test
	err := SaveTracks("test-user-id", testTracks, "test-source")

	// Assert the result
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error saving tracks")
}

func TestSaveTracks_BeginError(t *testing.T) {
	mockPool, _, _ := setupDBTest()
	defer teardownDBTest()

	// Create test tracks
	testTracks := []*Track{
		{
			TrackId: "track1",
			Name:    "Track 1",
		},
	}

	// Mock the getDB function
	getDB = func() (*pgxpool.Pool, error) {
		return mockPool, nil
	}

	// Mock the Begin function to return an error
	mockPool.On("Begin", context.Background()).Return(&MockTx{}, errors.New("begin error"))

	// Call the function under test
	err := SaveTracks("test-user-id", testTracks, "test-source")

	// Assert the result
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error saving tracks")
	mockPool.AssertExpectations(t)
}

func TestSaveTracks_BatchExecError(t *testing.T) {
	mockPool, mockTx, mockBatchResults := setupDBTest()
	defer teardownDBTest()

	// Create test tracks
	testTracks := []*Track{
		{
			TrackId: "track1",
			Name:    "Track 1",
		},
	}

	// Mock the getDB function
	getDB = func() (*pgxpool.Pool, error) {
		return mockPool, nil
	}

	// Mock the Begin function
	mockPool.On("Begin", context.Background()).Return(mockTx, nil)

	// Mock the SendBatch function
	mockTx.On("SendBatch", context.Background(), mock.Anything).Return(mockBatchResults)

	// Mock the Exec function for batch results to return an error
	mockBatchResults.On("Exec").Return(pgconn.CommandTag(""), errors.New("exec error"))

	// Mock the Close function for batch results
	mockBatchResults.On("Close").Return(nil)

	// Mock the Rollback function
	mockTx.On("Rollback", context.Background()).Return(nil)

	// Call the function under test
	err := SaveTracks("test-user-id", testTracks, "test-source")

	// Assert the result
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error saving tracks")
	mockPool.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	mockBatchResults.AssertExpectations(t)
}

func TestSaveTracks_CommitError(t *testing.T) {
	mockPool, mockTx, mockBatchResults := setupDBTest()
	defer teardownDBTest()

	// Create test tracks
	testTracks := []*Track{
		{
			TrackId: "track1",
			Name:    "Track 1",
		},
	}

	// Mock the getDB function
	getDB = func() (*pgxpool.Pool, error) {
		return mockPool, nil
	}

	// Mock the Begin function
	mockPool.On("Begin", context.Background()).Return(mockTx, nil)

	// Mock the SendBatch function
	mockTx.On("SendBatch", context.Background(), mock.Anything).Return(mockBatchResults)

	// Mock the Exec function for batch results
	mockCommandTag := pgconn.CommandTag("INSERT 1")
	mockBatchResults.On("Exec").Return(mockCommandTag, nil).Times(2) // 1 track * 2 queries (track + relation)

	// Mock the Close function for batch results
	mockBatchResults.On("Close").Return(nil).Times(2) // 2 batches (tracks + relations)

	// Mock the Commit function to return an error
	mockTx.On("Commit", context.Background()).Return(errors.New("commit error"))

	// Mock the Rollback function
	mockTx.On("Rollback", context.Background()).Return(nil)

	// Call the function under test
	err := SaveTracks("test-user-id", testTracks, "test-source")

	// Assert the result
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error saving tracks")
	mockPool.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	mockBatchResults.AssertExpectations(t)
}
