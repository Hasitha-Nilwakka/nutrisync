// This file contains unit tests for NormalizeWeightToKilograms, NormalizeHeightToCentimeters,
// and CalculateAndValidateBMI in package health.
package health

import (
	"testing"
)

// TestNormalizeWeightToKilogramsConvertsKgAndLbsAndRejectsInvalidRanges verifies metric/imperial weight
// conversions and boundary enforcement.
func TestNormalizeWeightToKilogramsConvertsKgAndLbsAndRejectsInvalidRanges(testingInstance *testing.T) {
	weightFromKg, errKg := NormalizeWeightToKilograms(75.5, "kg")
	if errKg != nil || weightFromKg != 75.5 {
		testingInstance.Fatalf("expected 75.5 kg, got %.2f (err=%v)", weightFromKg, errKg)
	}

	// 154.32 lbs ~= 70.00 kg
	weightFromLbs, errLbs := NormalizeWeightToKilograms(154.32, "lbs")
	if errLbs != nil || weightFromLbs != 70.0 {
		testingInstance.Fatalf("expected 70.00 kg from 154.32 lbs, got %.2f (err=%v)", weightFromLbs, errLbs)
	}

	invalidWeights := []struct {
		value float64
		unit  string
	}{
		{value: 0, unit: "kg"},
		{value: -65, unit: "kg"},
		{value: 10, unit: "kg"},
		{value: 500, unit: "kg"},
		{value: 70, unit: "stones"},
	}

	for _, testCase := range invalidWeights {
		if _, err := NormalizeWeightToKilograms(testCase.value, testCase.unit); err == nil {
			testingInstance.Errorf("expected error for weight=%v unit=%q, got nil", testCase.value, testCase.unit)
		}
	}
}

// TestNormalizeHeightToCentimetersConvertsCmAndInAndRejectsInvalidRanges verifies metric/imperial height
// conversions and boundary enforcement.
func TestNormalizeHeightToCentimetersConvertsCmAndInAndRejectsInvalidRanges(testingInstance *testing.T) {
	heightFromCm, errCm := NormalizeHeightToCentimeters(180.0, "cm")
	if errCm != nil || heightFromCm != 180.0 {
		testingInstance.Fatalf("expected 180.0 cm, got %.2f (err=%v)", heightFromCm, errCm)
	}

	// 70 inches == 177.80 cm
	heightFromIn, errIn := NormalizeHeightToCentimeters(70.0, "in")
	if errIn != nil || heightFromIn != 177.8 {
		testingInstance.Fatalf("expected 177.80 cm from 70 in, got %.2f (err=%v)", heightFromIn, errIn)
	}

	invalidHeights := []struct {
		value float64
		unit  string
	}{
		{value: 0, unit: "cm"},
		{value: -170, unit: "cm"},
		{value: 50, unit: "cm"},
		{value: 300, unit: "cm"},
		{value: 170, unit: "meters"},
	}

	for _, testCase := range invalidHeights {
		if _, err := NormalizeHeightToCentimeters(testCase.value, testCase.unit); err == nil {
			testingInstance.Errorf("expected error for height=%v unit=%q, got nil", testCase.value, testCase.unit)
		}
	}
}

// TestCalculateAndValidateBMIClassifiesAndRejectsUnrealisticValues verifies BMI calculation,
// classification categories, and unrealistic value rejection.
func TestCalculateAndValidateBMIClassifiesAndRejectsUnrealisticValues(testingInstance *testing.T) {
	bmi, classification, err := CalculateAndValidateBMI(72.0, 180.0)
	if err != nil {
		testingInstance.Fatalf("expected valid BMI, got error: %v", err)
	}

	if bmi != 22.22 || classification != "normal weight" {
		testingInstance.Errorf("expected BMI=22.22 classification='normal weight', got BMI=%.2f classification=%q", bmi, classification)
	}

	if _, classUnder, errUnder := CalculateAndValidateBMI(50.0, 180.0); errUnder != nil || classUnder != "underweight" {
		testingInstance.Errorf("expected underweight classification, got class=%q err=%v", classUnder, errUnder)
	}

	if _, classOver, errOver := CalculateAndValidateBMI(85.0, 175.0); errOver != nil || classOver != "overweight" {
		testingInstance.Errorf("expected overweight classification, got class=%q err=%v", classOver, errOver)
	}

	if _, classObese, errObese := CalculateAndValidateBMI(110.0, 175.0); errObese != nil || classObese != "obese" {
		testingInstance.Errorf("expected obese classification, got class=%q err=%v", classObese, errObese)
	}

	if _, _, unrealisticErr := CalculateAndValidateBMI(20.0, 220.0); unrealisticErr == nil {
		testingInstance.Fatal("expected error for unrealistic low BMI (< 10)")
	}

	if _, _, negErr := CalculateAndValidateBMI(-70.0, 175.0); negErr == nil {
		testingInstance.Fatal("expected error for negative weight")
	}
}
