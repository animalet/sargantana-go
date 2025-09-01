package controller

import (
	"errors"
	"testing"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-gonic/gin"
)

// MockController is a test implementation of IController for testing purposes
type MockController struct {
	bindCalled  bool
	closeCalled bool
	shouldError bool
}

func (m *MockController) Bind(_ *gin.Engine, _ gin.HandlerFunc) {
	m.bindCalled = true
}

func (m *MockController) Close() error {
	m.closeCalled = true
	if m.shouldError {
		return errors.New("mock close error")
	}
	return nil
}

// MockControllerFactory creates a new MockController instance
func MockControllerFactory(_ config.ControllerConfig, _ config.ServerConfig) (IController, error) {
	return &MockController{}, nil
}

// ErrorControllerFactory always returns an error for testing error scenarios
func ErrorControllerFactory(_ config.ControllerConfig, _ config.ServerConfig) (IController, error) {
	return nil, errors.New("factory error")
}

func TestRegisterController(t *testing.T) {
	// Clear registry before test
	originalRegistry := controllerRegistry
	controllerRegistry = make(map[string]NewController)
	defer func() {
		controllerRegistry = originalRegistry
	}()

	tests := []struct {
		name     string
		typeName string
		factory  NewController
	}{
		{
			name:     "register valid controller",
			typeName: "mock",
			factory:  MockControllerFactory,
		},
		{
			name:     "register controller with special characters",
			typeName: "test-controller_v1",
			factory:  MockControllerFactory,
		},
		{
			name:     "register controller with empty name",
			typeName: "",
			factory:  MockControllerFactory,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterController(tt.typeName, tt.factory)

			if factory, exists := controllerRegistry[tt.typeName]; !exists {
				t.Errorf("Controller %s was not registered", tt.typeName)
			} else if factory == nil {
				t.Errorf("Registered factory for %s is nil", tt.typeName)
			}
		})
	}
}

func TestGetControllerFactory(t *testing.T) {
	// Clear registry before test
	originalRegistry := controllerRegistry
	controllerRegistry = make(map[string]NewController)
	defer func() {
		controllerRegistry = originalRegistry
	}()

	// Register a test controller
	RegisterController("test", MockControllerFactory)

	tests := []struct {
		name         string
		typeName     string
		expectExists bool
	}{
		{
			name:         "get existing controller",
			typeName:     "test",
			expectExists: true,
		},
		{
			name:         "get non-existing controller",
			typeName:     "nonexistent",
			expectExists: false,
		},
		{
			name:         "get controller with empty name",
			typeName:     "",
			expectExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, exists := GetControllerFactory(tt.typeName)

			if exists != tt.expectExists {
				t.Errorf("GetControllerFactory() exists = %v, want %v", exists, tt.expectExists)
			}

			if tt.expectExists && factory == nil {
				t.Errorf("Expected factory to be non-nil for existing controller %s", tt.typeName)
			}

			if !tt.expectExists && factory != nil {
				t.Errorf("Expected factory to be nil for non-existing controller %s", tt.typeName)
			}
		})
	}
}

func TestControllerRegistry_Overwrite(t *testing.T) {
	// Clear registry before test
	originalRegistry := controllerRegistry
	controllerRegistry = make(map[string]NewController)
	defer func() {
		controllerRegistry = originalRegistry
	}()

	typeName := "overwrite-test"

	// Register first factory
	RegisterController(typeName, MockControllerFactory)
	factory1, exists1 := GetControllerFactory(typeName)
	if !exists1 {
		t.Fatal("First registration failed")
	}

	// Register second factory (should overwrite)
	RegisterController(typeName, ErrorControllerFactory)
	factory2, exists2 := GetControllerFactory(typeName)
	if !exists2 {
		t.Fatal("Second registration failed")
	}

	// Verify that the factory was overwritten by testing the behavior
	_, err1 := factory1(nil, config.ServerConfig{})
	_, err2 := factory2(nil, config.ServerConfig{})

	if err1 != nil {
		t.Errorf("First factory should not return error, got: %v", err1)
	}
	if err2 == nil {
		t.Error("Second factory should return error, got nil")
	}
}

func TestMockController_Interface(t *testing.T) {
	// Verify MockController implements IController interface
	var _ IController = &MockController{}

	mock := &MockController{}

	// Test Bind method
	engine := gin.New()
	middleware := func(c *gin.Context) {}

	if mock.bindCalled {
		t.Error("bindCalled should be false initially")
	}

	mock.Bind(engine, middleware)

	if !mock.bindCalled {
		t.Error("bindCalled should be true after calling Bind")
	}

	// Test Close method without error
	if mock.closeCalled {
		t.Error("closeCalled should be false initially")
	}

	err := mock.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got: %v", err)
	}

	if !mock.closeCalled {
		t.Error("closeCalled should be true after calling Close")
	}
}

func TestMockController_CloseError(t *testing.T) {
	mock := &MockController{shouldError: true}

	err := mock.Close()
	if err == nil {
		t.Error("Close() should return error when shouldError is true")
		return
	}

	expectedError := "mock close error"
	if err.Error() != expectedError {
		t.Errorf("Close() error = %v, want %v", err.Error(), expectedError)
	}
}

func TestControllerFactory_Success(t *testing.T) {
	controllerConfig := config.ControllerConfig{}
	serverConfig := config.ServerConfig{
		Address: "localhost:8080",
	}

	controller, err := MockControllerFactory(controllerConfig, serverConfig)

	if err != nil {
		t.Errorf("MockControllerFactory should not return error, got: %v", err)
	}

	if controller == nil {
		t.Error("MockControllerFactory should return non-nil controller")
	}
}

func TestControllerFactory_Error(t *testing.T) {
	controllerConfig := config.ControllerConfig{}
	serverConfig := config.ServerConfig{}

	controller, err := ErrorControllerFactory(controllerConfig, serverConfig)

	if err == nil {
		t.Error("ErrorControllerFactory should return error")
		return
	}

	if controller != nil {
		t.Error("ErrorControllerFactory should return nil controller when error occurs")
	}

	expectedError := "factory error"
	if err.Error() != expectedError {
		t.Errorf("ErrorControllerFactory error = %v, want %v", err.Error(), expectedError)
	}
}

func TestControllerRegistry_ConcurrentAccess(t *testing.T) {
	// Clear registry before test
	originalRegistry := controllerRegistry
	controllerRegistry = make(map[string]NewController)
	defer func() {
		controllerRegistry = originalRegistry
	}()

	// Test concurrent registration and retrieval
	done := make(chan bool, 2)

	// Goroutine 1: Register controllers
	go func() {
		for i := 0; i < 100; i++ {
			RegisterController("concurrent1", MockControllerFactory)
		}
		done <- true
	}()

	// Goroutine 2: Get controllers
	go func() {
		for i := 0; i < 100; i++ {
			GetControllerFactory("concurrent1")
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify final state
	factory, exists := GetControllerFactory("concurrent1")
	if !exists {
		t.Error("Controller should exist after concurrent operations")
	}
	if factory == nil {
		t.Error("Factory should not be nil after concurrent operations")
	}
}
