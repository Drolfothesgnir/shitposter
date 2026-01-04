package scum

// Probe is a special kind of step which performs checks on the current state of the
// handling.
type Probe func(*ActionContext) bool

// StepProbeAndDecide performs checks on the current state of handling, possibly mutates the
// provided [ActionContext] and calls successStep if the checks passed and failureStep otherwise.
func StepProbeAndDecide(probeStep Probe, successStep, failureStep Step) Step {
	return func(ac *ActionContext) bool {
		if probeStep(ac) {
			return successStep(ac)
		}

		return failureStep(ac)
	}
}
