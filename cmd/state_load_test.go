package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestLoadActiveColonyStateNormalizesLegacyPausedState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Normalize old paused colonies"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.State("PAUSED"),
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Legacy paused phase", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: &taskID, Goal: "Resume safely", Status: colony.TaskInProgress}}},
			},
		},
	})

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState returned error: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY", state.State)
	}
	if !state.Paused {
		t.Fatal("expected legacy PAUSED state to normalize with paused flag set")
	}
}

func TestLoadActiveColonyStateNormalizesBrokenIdleStateWithGoal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Recover broken idle colony"
	taskID := "task-1"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateIDLE,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Restorable phase", Status: colony.PhaseInProgress, Tasks: []colony.Task{{ID: &taskID, Goal: "Finish restoring", Status: colony.TaskInProgress}}},
			},
		},
	})

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState returned error: %v", err)
	}
	if state.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY", state.State)
	}
}

func TestLoadActiveColonyStateRepairsMissingPlanFromPlanningArtifact(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Repair a colony that lost plan.phases"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		Plan:         colony.Plan{Phases: []colony.Phase{}},
		Events: []string{
			"2026-04-21T07:50:00Z phase-1-complete: Audit complete.",
			"2026-04-21T08:20:00Z phase-2-complete: Standard designed.",
		},
	})
	if err := store.SaveJSON("planning/phase-plan.json", codexWorkerPlanArtifact{
		Confidence: codexPlanConfidence{Overall: 82},
		Phases: []codexWorkerPlanPhase{
			{Name: "Audit", Tasks: []codexWorkerPlanTask{{Goal: "Complete the audit"}}},
			{Name: "Design", Tasks: []codexWorkerPlanTask{{Goal: "Design the standard"}}},
			{Name: "Rollout", Tasks: []codexWorkerPlanTask{{Goal: "Apply the schema"}}},
		},
	}); err != nil {
		t.Fatalf("failed to save planning artifact: %v", err)
	}

	state, err := loadActiveColonyState()
	if err != nil {
		t.Fatalf("loadActiveColonyState returned error: %v", err)
	}
	if len(state.Plan.Phases) != 3 {
		t.Fatalf("repaired plan phase count = %d, want 3", len(state.Plan.Phases))
	}
	if state.CurrentPhase != 3 {
		t.Fatalf("current_phase = %d, want 3", state.CurrentPhase)
	}
	if state.Plan.GeneratedAt == nil {
		t.Fatal("expected repaired plan generated_at to be set")
	}
	if state.Plan.Confidence == nil || *state.Plan.Confidence != 0.82 {
		t.Fatalf("plan confidence = %v, want 0.82", state.Plan.Confidence)
	}
	if state.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Fatalf("phase 1 status = %s, want completed", state.Plan.Phases[0].Status)
	}
	if state.Plan.Phases[1].Status != colony.PhaseCompleted {
		t.Fatalf("phase 2 status = %s, want completed", state.Plan.Phases[1].Status)
	}
	if state.Plan.Phases[2].Status != colony.PhaseReady {
		t.Fatalf("phase 3 status = %s, want ready", state.Plan.Phases[2].Status)
	}
	if len(state.Events) == 0 || !strings.Contains(state.Events[len(state.Events)-1], "plan_recovered|state|Recovered 3 phases") {
		t.Fatalf("expected repair event, got %v", state.Events)
	}

	var persisted colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &persisted); err != nil {
		t.Fatalf("failed to reload repaired state: %v", err)
	}
	if len(persisted.Plan.Phases) != 3 || persisted.CurrentPhase != 3 {
		t.Fatalf("persisted state was not repaired: %+v", persisted)
	}
}
