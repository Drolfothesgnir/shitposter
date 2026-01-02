package scum

// ProbeStep is a special kind of step which performs checks on the current state of the
// handling.
//
// WARNING: ProbeStep may mutate the ActionContext and return true only if the checks passed and
// not when the case was handled, since it doesn't handle anything.
type ProbeStep func(*ActionContext) bool

// StepProbeAndDecide performs checks on the current state of handling, possibly mutates the
// provided [ActionContext] and calls successStep if the checks passed and failureStep otherwise.
func StepProbeAndDecide(probeStep ProbeStep, successStep, failureStep Step) Step {
	return func(ac *ActionContext) bool {
		if probeStep(ac) {
			return successStep(ac)
		}

		return failureStep(ac)
	}
}
