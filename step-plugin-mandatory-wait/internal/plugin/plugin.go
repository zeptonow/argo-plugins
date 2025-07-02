package plugin

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/argoproj/argo-rollouts/rollout/steps/plugin/rpc"
	"github.com/argoproj/argo-rollouts/utils/plugin/types"
)

// Config defines the configuration for the mandatory wait plugin
type Config struct {
	// Duration specifies how long to block the rollout (e.g., "5m", "30s", "1h")
	Duration string `json:"duration"`
}

// State tracks the plugin's internal state
type State struct {
	// ID is a unique identifier for this step execution
	ID string `json:"id"`
	// StartTime is when the mandatory wait period began
	StartTime *time.Time `json:"startTime,omitempty"`
	// Duration is the parsed duration from config
	Duration time.Duration `json:"duration"`
	// Completed indicates if the mandatory wait has finished
	Completed bool `json:"completed"`
}

type rpcPlugin struct {
	LogCtx *log.Entry
}

func New(logCtx *log.Entry) rpc.StepPlugin {
	return &rpcPlugin{
		LogCtx: logCtx,
	}
}

func (p *rpcPlugin) InitPlugin() types.RpcError {
	p.LogCtx.Info("InitPlugin: mandatory-wait plugin initialized")
	return types.RpcError{}
}

func (p *rpcPlugin) Run(rollout *v1alpha1.Rollout, context *types.RpcStepContext) (types.RpcStepResult, types.RpcError) {
	p.LogCtx.Infof("Run: mandatory-wait plugin for rollout %s/%s", rollout.Namespace, rollout.Name)

	// Parse config and state
	var config Config
	var state State
	if context != nil {
		if context.Config != nil {
			if err := json.Unmarshal(context.Config, &config); err != nil {
				return types.RpcStepResult{}, types.RpcError{ErrorString: fmt.Errorf("could not unmarshal config: %w", err).Error()}
			}
			p.LogCtx.Infof("Using config: %+v", config)
		}
		if context.Status != nil {
			if err := json.Unmarshal(context.Status, &state); err != nil {
				return types.RpcStepResult{}, types.RpcError{ErrorString: fmt.Errorf("could not unmarshal status: %w", err).Error()}
			}
			p.LogCtx.Infof("Existing state: %+v", state)
		}
	}

	// Validate configuration
	if config.Duration == "" {
		return types.RpcStepResult{}, types.RpcError{ErrorString: "duration is required in config"}
	}

	duration, err := time.ParseDuration(config.Duration)
	if err != nil {
		return types.RpcStepResult{}, types.RpcError{ErrorString: fmt.Errorf("invalid duration format: %w", err).Error()}
	}

	// Initialize state if this is the first run
	if state.ID == "" {
		now := time.Now()
		state.ID = uuid.New().String()
		state.StartTime = &now
		state.Duration = duration
		state.Completed = false
		p.LogCtx.Infof("Starting mandatory wait period of %s (ID: %s)", duration, state.ID)
	}

	// Check if already completed
	if state.Completed {
		p.LogCtx.Infof("Mandatory wait period already completed for ID: %s", state.ID)
		return p.completedResult(state, "Mandatory wait period completed")
	}

	// Check if wait period has elapsed
	if state.StartTime != nil {
		elapsed := time.Since(*state.StartTime)
		remaining := state.Duration - elapsed

		if remaining <= 0 {
			// Wait period is complete
			state.Completed = true
			p.LogCtx.Infof("Mandatory wait period completed after %s (ID: %s)", elapsed, state.ID)
			return p.completedResult(state, fmt.Sprintf("Mandatory wait period of %s completed", state.Duration))
		}

		// Still waiting
		p.LogCtx.Infof("Mandatory wait in progress: %s elapsed, %s remaining (ID: %s)", elapsed, remaining, state.ID)
		return p.runningResult(state, fmt.Sprintf("Mandatory wait: %s remaining", remaining.Round(time.Second)))
	}

	// This shouldn't happen, but handle gracefully
	return types.RpcStepResult{}, types.RpcError{ErrorString: "invalid state: startTime is nil"}
}

func (p *rpcPlugin) Terminate(rollout *v1alpha1.Rollout, context *types.RpcStepContext) (types.RpcStepResult, types.RpcError) {
	p.LogCtx.Infof("Terminate: mandatory-wait plugin for rollout %s/%s", rollout.Namespace, rollout.Name)

	// Parse state
	var state State
	if context != nil && context.Status != nil {
		if err := json.Unmarshal(context.Status, &state); err != nil {
			return types.RpcStepResult{}, types.RpcError{ErrorString: fmt.Errorf("could not unmarshal status: %w", err).Error()}
		}
	}

	// For mandatory wait, termination should still require the wait period to complete
	// This prevents bypassing the mandatory wait through termination
	if !state.Completed && state.StartTime != nil {
		elapsed := time.Since(*state.StartTime)
		remaining := state.Duration - elapsed

		if remaining > 0 {
			p.LogCtx.Warnf("Terminate called but mandatory wait period not completed: %s remaining", remaining)
			return p.runningResult(state, fmt.Sprintf("Mandatory wait cannot be terminated: %s remaining", remaining.Round(time.Second)))
		}
	}

	// Wait period completed, allow termination
	state.Completed = true
	p.LogCtx.Infof("Terminate: mandatory wait completed for ID: %s", state.ID)
	return p.completedResult(state, "Mandatory wait terminated (period completed)")
}

func (p *rpcPlugin) Abort(rollout *v1alpha1.Rollout, context *types.RpcStepContext) (types.RpcStepResult, types.RpcError) {
	p.LogCtx.Infof("Abort: mandatory-wait plugin for rollout %s/%s", rollout.Namespace, rollout.Name)

	// Parse state
	var state State
	if context != nil && context.Status != nil {
		if err := json.Unmarshal(context.Status, &state); err != nil {
			return types.RpcStepResult{}, types.RpcError{ErrorString: fmt.Errorf("could not unmarshal status: %w", err).Error()}
		}
	}

	// Mark as completed to allow abort
	state.Completed = true
	p.LogCtx.Infof("Abort: mandatory wait aborted for ID: %s", state.ID)
	return p.completedResult(state, "Mandatory wait aborted")
}

func (p *rpcPlugin) Type() string {
	return "mandatory-wait"
}

func (p *rpcPlugin) completedResult(state State, message string) (types.RpcStepResult, types.RpcError) {
	stateRaw, err := json.Marshal(state)
	if err != nil {
		return types.RpcStepResult{}, types.RpcError{ErrorString: fmt.Sprintf("Could not marshal state: %v", err)}
	}

	return types.RpcStepResult{
		Phase:   types.PhaseSuccessful,
		Message: message,
		Status:  stateRaw,
	}, types.RpcError{}
}

func (p *rpcPlugin) runningResult(state State, message string) (types.RpcStepResult, types.RpcError) {
	stateRaw, err := json.Marshal(state)
	if err != nil {
		return types.RpcStepResult{}, types.RpcError{ErrorString: fmt.Sprintf("Could not marshal state: %v", err)}
	}

	// Requeue every 15 seconds to check if wait period is complete
	return types.RpcStepResult{
		Phase:        types.PhaseRunning,
		Message:      message,
		RequeueAfter: 15 * time.Second,
		Status:       stateRaw,
	}, types.RpcError{}
}
