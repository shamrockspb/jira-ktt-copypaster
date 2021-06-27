package main

import (
	"testing"
)

func Test_validateApplicationMode(t *testing.T) {
	if err := validateApplicationMode("normal"); err != nil {
		t.Errorf("Result is incorrect, got: %v err, want: %v.", err, nil)
	}

	if err := validateApplicationMode("test"); err != nil {
		t.Errorf("Result is incorrect, got: %v err, want: %v.", err, nil)
	}

	if err := validateApplicationMode("incorrect_mode"); err == nil {
		t.Errorf("Result is incorrect, got: %v err, want: %v.", err, "Error" )
	}
	
}