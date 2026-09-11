package protocol

import (
	"testing"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	payload := WelcomePayload{
		NodeID:       2,
		AssignedTile: AssignedTile{Col: 2, Rows: []int{0, 1, 2}},
		CurrentScene: "three-unified",
		SceneEpochMs: 1755512300000,
		ServerTimeMs: 1755512345005,
		TickHz:       30,
	}
	raw, err := Marshal(TypeWelcome, 1, 1755512345005, payload)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	env, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if env.Type != TypeWelcome {
		t.Fatalf("Type: got %q want %q", env.Type, TypeWelcome)
	}
	if env.Seq != 1 {
		t.Fatalf("Seq: got %d want 1", env.Seq)
	}
	var got WelcomePayload
	if err := UnmarshalPayload(env, &got); err != nil {
		t.Fatalf("UnmarshalPayload: %v", err)
	}
	if got.NodeID != 2 || got.CurrentScene != "three-unified" || got.AssignedTile.Col != 2 {
		t.Fatalf("payload roundtrip mismatch: %+v", got)
	}
}
