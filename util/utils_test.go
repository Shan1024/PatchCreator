// Copyright (c) 2016, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.

package util

import (
	"testing"
	"github.com/wso2/wum-uc/constant"
)

func TestProcessUserPreferenceScenario01(t *testing.T) {
	preference := ProcessUserPreference("yes")
	if preference != constant.YES {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.YES, preference)
	}
	preference = ProcessUserPreference("Yes")
	if preference != constant.YES {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.YES, preference)
	}
	preference = ProcessUserPreference("YES")
	if preference != constant.YES {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.YES, preference)
	}
	preference = ProcessUserPreference("y")
	if preference != constant.YES {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.YES, preference)
	}
	preference = ProcessUserPreference("Y")
	if preference != constant.YES {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.YES, preference)
	}
}

func TestProcessUserPreferenceScenario02(t *testing.T) {
	preference := ProcessUserPreference("no")
	if preference != constant.NO {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.NO, preference)
	}
	preference = ProcessUserPreference("No")
	if preference != constant.NO {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.NO, preference)
	}
	preference = ProcessUserPreference("NO")
	if preference != constant.NO {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.NO, preference)
	}
	preference = ProcessUserPreference("n")
	if preference != constant.NO {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.NO, preference)
	}
	preference = ProcessUserPreference("N")
	if preference != constant.NO {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.NO, preference)
	}
}

func TestProcessUserPreferenceScenario03(t *testing.T) {
	preference := ProcessUserPreference("reenter")
	if preference != constant.REENTER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.REENTER, preference)
	}
	preference = ProcessUserPreference("re-enter")
	if preference != constant.REENTER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.REENTER, preference)
	}
	preference = ProcessUserPreference("Reenter")
	if preference != constant.REENTER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.REENTER, preference)
	}
	preference = ProcessUserPreference("Re-enter")
	if preference != constant.REENTER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.REENTER, preference)
	}
	preference = ProcessUserPreference("REENTER")
	if preference != constant.REENTER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.REENTER, preference)
	}
	preference = ProcessUserPreference("RE-ENTER")
	if preference != constant.REENTER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.REENTER, preference)
	}
	preference = ProcessUserPreference("r")
	if preference != constant.REENTER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.REENTER, preference)
	}
	preference = ProcessUserPreference("R")
	if preference != constant.REENTER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.REENTER, preference)
	}
}

func TestProcessUserPreferenceScenario04(t *testing.T) {
	preference := ProcessUserPreference("ya")
	if preference != constant.OTHER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.OTHER, preference)
	}
	preference = ProcessUserPreference("nope")
	if preference != constant.OTHER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.OTHER, preference)
	}
	preference = ProcessUserPreference("re")
	if preference != constant.OTHER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.OTHER, preference)
	}
	preference = ProcessUserPreference("random")
	if preference != constant.OTHER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.OTHER, preference)
	}
	preference = ProcessUserPreference("1234")
	if preference != constant.OTHER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.OTHER, preference)
	}
	preference = ProcessUserPreference("abc")
	if preference != constant.OTHER {
		t.Errorf("Test failed, expected: %d, actual: %d", constant.OTHER, preference)
	}
}

func TestIsUserPreferencesValid01(t *testing.T) {
	preferences := []string{"3", "1", "2"}
	isValid, err := IsUserPreferencesValid(preferences, len(preferences))
	if err != nil {
		t.Errorf("Test failed. Unexpected error: %v", err)
	}
	if !isValid {
		t.Errorf("Test failed, expected: %v, actual: %v", true, isValid)
	}

	preferences = []string{"0"}
	isValid, err = IsUserPreferencesValid(preferences, len(preferences))
	if err != nil {
		t.Errorf("Test failed. Unexpected error: %v", err)
	}
	if !isValid {
		t.Errorf("Test failed, expected: %v, actual: %v", true, isValid)
	}

	preferences = []string{"-1"}
	isValid, err = IsUserPreferencesValid(preferences, len(preferences))
	if err == nil {
		t.Error("Test failed. Error expected")
	}
	if isValid {
		t.Errorf("Test failed, expected: %v, actual: %v", false, isValid)
	}

	preferences = []string{"10"}
	isValid, err = IsUserPreferencesValid(preferences, len(preferences))
	if err == nil {
		t.Error("Test failed. Error expected")
	}
	if isValid {
		t.Errorf("Test failed, expected: %v, actual: %v", false, isValid)
	}
}
