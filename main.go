package main

import (
	"fmt"
	"time"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/argoproj/argo-rollouts/rollout/trafficrouting/plugin/rpc"
	"github.com/argoproj/argo-rollouts/utils/plugin/types"
	goPlugin "github.com/hashicorp/go-plugin"
	log "github.com/sirupsen/logrus"
)

// MandatoryPausePlugin implements a step plugin that enforces mandatory pauses
type MandatoryPausePlugin struct {
	LogCtx *log.Entry
}

// Type returns the type of the plugin
func (p *MandatoryPausePlugin) Type() string {
	return "MandatoryPause"
}

// InitPlugin initializes the plugin
func (p *MandatoryPausePlugin) InitPlugin() types.RpcError {
	p.LogCtx = log.WithFields(log.Fields{
		"plugin": "mandatory-pause",
	})
	p.LogCtx.Info("Mandatory pause plugin initialized")
	return types.RpcError{}
}

// Run executes the mandatory pause logic
func (p *MandatoryPausePlugin) Run(rollout *v1alpha1.Rollout, context *types.RpcStepContext) (types.RpcStepResult, types.RpcError) {
	if p.LogCtx == nil {
		p.LogCtx = log.WithFields(log.Fields{
			"rollout":   rollout.Name,
			"namespace": rollout.Namespace,
			"plugin":    "mandatory-pause",
		})
	}

	// Get pause duration from plugin configuration
	pauseDuration := "2m" // Default 2 minutes
	if duration, ok := context.Config["duration"].(string); ok {
		pauseDuration = duration
	}

	// Parse duration
	duration, err := time.ParseDuration(pauseDuration)
	if err != nil {
		return types.RpcStepResult{}, types.RpcError{ErrorString: fmt.Sprintf("invalid pause duration: %v", err)}
	}

	// Check if this is the first time running this step
	startTimeKey := fmt.Sprintf("pause-start-%d", context.StepIndex)

	// Get or set the pause start time
	startTimeStr, exists := context.State[startTimeKey]
	var startTime time.Time

	if !exists {
		// First time running this step - record start time
		startTime = time.Now()
		if context.State == nil {
			context.State = make(map[string]string)
		}
		context.State[startTimeKey] = startTime.Format(time.RFC3339)
		p.LogCtx.Infof("Starting mandatory pause for %s", pauseDuration)
	} else {
		// Parse existing start time
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return types.RpcStepResult{}, types.RpcError{ErrorString: fmt.Sprintf("failed to parse start time: %v", err)}
		}
	}

	// Calculate elapsed time
	elapsed := time.Since(startTime)
	remaining := duration - elapsed

	// Check if pause is complete
	if remaining <= 0 {
		p.LogCtx.Info("Mandatory pause completed")
		// Clean up state
		delete(context.State, startTimeKey)

		return types.RpcStepResult{
			Phase:   types.PhaseSuccessful,
			Message: fmt.Sprintf("Mandatory pause of %s completed", pauseDuration),
		}, types.RpcError{}
	}

	// Pause still in progress
	p.LogCtx.Infof("Pause in progress: %s remaining", remaining.Round(time.Second))

	return types.RpcStepResult{
		Phase:   types.PhaseRunning,
		Message: fmt.Sprintf("Mandatory pause: %s remaining", remaining.Round(time.Second)),
		RequeueAfter: &types.Duration{
			Duration: time.Second * 10, // Check again in 10 seconds
		},
	}, types.RpcError{}
}

// Terminate handles cleanup when the step is terminated
func (p *MandatoryPausePlugin) Terminate(rollout *v1alpha1.Rollout, context *types.RpcStepContext) (types.RpcStepResult, types.RpcError) {
	if p.LogCtx == nil {
		p.LogCtx = log.WithFields(log.Fields{
			"rollout":   rollout.Name,
			"namespace": rollout.Namespace,
			"plugin":    "mandatory-pause",
		})
	}

	p.LogCtx.Info("Terminating mandatory pause")

	// Clean up any state
	startTimeKey := fmt.Sprintf("pause-start-%d", context.StepIndex)
	if context.State != nil {
		delete(context.State, startTimeKey)
	}

	return types.RpcStepResult{
		Phase:   types.PhaseFailed,
		Message: "Mandatory pause was terminated",
	}, types.RpcError{}
}

// Abort handles cleanup when the rollout is aborted
func (p *MandatoryPausePlugin) Abort(rollout *v1alpha1.Rollout, context *types.RpcStepContext) (types.RpcStepResult, types.RpcError) {
	if p.LogCtx == nil {
		p.LogCtx = log.WithFields(log.Fields{
			"rollout":   rollout.Name,
			"namespace": rollout.Namespace,
			"plugin":    "mandatory-pause",
		})
	}

	p.LogCtx.Info("Aborting mandatory pause")

	// Clean up any state
	startTimeKey := fmt.Sprintf("pause-start-%d", context.StepIndex)
	if context.State != nil {
		delete(context.State, startTimeKey)
	}

	return types.RpcStepResult{
		Phase:   types.PhaseFailed,
		Message: "Mandatory pause was aborted",
	}, types.RpcError{}
}

func main() {
	handshakeConfig := goPlugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "ARGO_ROLLOUTS_RPC_PLUGIN",
		MagicCookieValue: "step",
	}

	pluginMap := map[string]goPlugin.Plugin{
		"RpcStepPlugin": &rpc.RpcStepPlugin{Impl: &MandatoryPausePlugin{}},
	}

	goPlugin.Serve(&goPlugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
