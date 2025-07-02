package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	goPlugin "github.com/hashicorp/go-plugin"
	log "github.com/sirupsen/logrus"

	"argo-rollouts-mandatory-pause-plugin/step-plugin-mandatory-wait/internal/plugin"

	"github.com/argoproj/argo-rollouts/rollout/steps/plugin/rpc"
	rolloutsPlugin "github.com/argoproj/argo-rollouts/rollout/steps/plugin/rpc"
	"github.com/argoproj/argo-rollouts/utils/plugin/types"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
)

var testHandshake = goPlugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "ARGO_ROLLOUTS_RPC_PLUGIN",
	MagicCookieValue: "step",
}

func pluginClient(t *testing.T) (rpc.StepPlugin, goPlugin.ClientProtocol, func(), chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())

	pluginImpl := plugin.New(log.WithFields(log.Fields{}))

	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]goPlugin.Plugin{
		"RpcStepPlugin": &rolloutsPlugin.RpcStepPlugin{Impl: pluginImpl},
	}

	ch := make(chan *goPlugin.ReattachConfig, 1)
	closeCh := make(chan struct{})
	go goPlugin.Serve(&goPlugin.ServeConfig{
		HandshakeConfig: testHandshake,
		Plugins:         pluginMap,
		Test: &goPlugin.ServeTestConfig{
			Context:          ctx,
			ReattachConfigCh: ch,
			CloseCh:          closeCh,
		},
	})

	// We should get a config
	var config *goPlugin.ReattachConfig
	select {
	case config = <-ch:
	case <-time.After(2000 * time.Millisecond):
		t.Fatal("should've received reattach")
	}
	if config == nil {
		t.Fatal("config should not be nil")
	}

	// Connect!
	c := goPlugin.NewClient(&goPlugin.ClientConfig{
		Cmd:             nil,
		HandshakeConfig: testHandshake,
		Plugins:         pluginMap,
		Reattach:        config,
	})
	client, err := c.Client()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Request the plugin
	raw, err := client.Dispense("RpcStepPlugin")
	if err != nil {
		t.Fail()
	}

	plugin, ok := raw.(rpc.StepPlugin)
	if !ok {
		t.Fail()
	}

	return plugin, client, cancel, closeCh
}

func TestMandatoryWaitPlugin(t *testing.T) {
	plugin, _, cancel, closeCh := pluginClient(t)
	defer cancel()

	err := plugin.InitPlugin()
	assert.Equal(t, "", err.Error(), "InitPlugin should not return an error")

	ro := v1alpha1.Rollout{}

	// Test with valid configuration
	configData, _ := json.Marshal(map[string]string{"duration": "5s"})
	rpcCtx := &types.RpcStepContext{
		PluginName: "mandatory-wait",
		Config:     configData,
		Status:     nil,
	}

	// First run should return Running status
	result, err := plugin.Run(&ro, rpcCtx)
	assert.Equal(t, "", err.Error(), "Run should not return an error")
	assert.Equal(t, types.PhaseRunning, result.Phase, "First run should return Running phase")
	assert.NotNil(t, result.Status, "Status should not be nil")
	assert.Equal(t, 15*time.Second, result.RequeueAfter, "Should requeue after 15 seconds")

	// Simulate running with existing state (should still be running)
	rpcCtx.Status = result.Status
	result2, err := plugin.Run(&ro, rpcCtx)
	assert.Equal(t, "", err.Error(), "Second run should not return an error")
	assert.Equal(t, types.PhaseRunning, result2.Phase, "Should still be running")

	// Canceling should cause an exit
	cancel()
	<-closeCh
}

func TestMandatoryWaitPluginInvalidConfig(t *testing.T) {
	plugin, _, cancel, closeCh := pluginClient(t)
	defer cancel()

	err := plugin.InitPlugin()
	assert.Equal(t, "", err.Error(), "InitPlugin should not return an error")

	ro := v1alpha1.Rollout{}

	// Test with missing duration
	configData, _ := json.Marshal(map[string]string{})
	rpcCtx := &types.RpcStepContext{
		PluginName: "mandatory-wait",
		Config:     configData,
		Status:     nil,
	}

	_, err = plugin.Run(&ro, rpcCtx)
	assert.NotEqual(t, "", err.Error(), "Should return an error for missing duration")

	// Test with invalid duration format
	configData, _ = json.Marshal(map[string]string{"duration": "invalid"})
	rpcCtx.Config = configData

	_, err = plugin.Run(&ro, rpcCtx)
	assert.NotEqual(t, "", err.Error(), "Should return an error for invalid duration format")

	// Canceling should cause an exit
	cancel()
	<-closeCh
}

func TestMandatoryWaitPluginTerminate(t *testing.T) {
	plugin, _, cancel, closeCh := pluginClient(t)
	defer cancel()

	err := plugin.InitPlugin()
	assert.Equal(t, "", err.Error(), "InitPlugin should not return an error")

	ro := v1alpha1.Rollout{}

	// Start a wait period
	configData, _ := json.Marshal(map[string]string{"duration": "10s"})
	rpcCtx := &types.RpcStepContext{
		PluginName: "mandatory-wait",
		Config:     configData,
		Status:     nil,
	}

	result, err := plugin.Run(&ro, rpcCtx)
	assert.Equal(t, "", err.Error(), "Run should not return an error")
	assert.Equal(t, types.PhaseRunning, result.Phase, "Should be running")

	// Try to terminate while still waiting - should still be running
	rpcCtx.Status = result.Status
	termResult, err := plugin.Terminate(&ro, rpcCtx)
	assert.Equal(t, "", err.Error(), "Terminate should not return an error")
	assert.Equal(t, types.PhaseRunning, termResult.Phase, "Should still be running even after terminate")

	// Canceling should cause an exit
	cancel()
	<-closeCh
}
