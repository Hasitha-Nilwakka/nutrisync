// Package health provides unit conversion (lbs -> kg, in -> cm), range validation,
// and Body Mass Index (BMI) calculation and classification helpers.
package health

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

const (
	kilogramsPerPound     = 0.45359237
	centimetersPerInch    = 2.54
	minimumValidWeightKg  = 20.0
	maximumValidWeightKg  = 350.0
	minimumValidHeightCm  = 80.0
	maximumValidHeightCm  = 260.0
	minimumValidBMIValue  = 10.0
	maximumValidBMIValue  = 80.0
	underweightBMILimit   = 18.5
	normalWeightBMILimit  = 25.0
	overweightBMILimit    = 30.0
)

// roundToTwoDecimalPlaces rounds a floating-point number to 2 decimal places.
func roundToTwoDecimalPlaces(value float64) float64 {
	return math.Round(value*100.0) / 100.0
}

// NormalizeWeightToKilograms converts a weight measurement in "kg" or "lbs" into kilograms rounded to 2 decimal places,
// rejecting non-positive inputs or normalized values outside [20.0, 350.0] kg.
func NormalizeWeightToKilograms(value float64, unit string) (float64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0, errors.New("weight must be a positive number")
	}

	normalizedUnit := strings.ToLower(strings.TrimSpace(unit))
	if normalizedUnit == "" {
		normalizedUnit = "kg"
	}

	var weightKg float64
	switch normalizedUnit {
	case "kg", "kgs", "kilogram", "kilograms":
		weightKg = value
	case "lb", "lbs", "pound", "pounds":
		weightKg = value * kilogramsPerPound
	default:
		return 0, fmt.Errorf("unsupported weight unit %q (expected 'kg' or 'lbs')", unit)
	}

	roundedWeightKg := roundToTwoDecimalPlaces(weightKg)
	if roundedWeightKg < minimumValidWeightKg || roundedWeightKg > maximumValidWeightKg {
		return 0, fmt.Errorf(
			"normalized weight %.2f kg is outside the valid range [%.1f, %.1f] kg",
			roundedWeightKg,
			minimumValidWeightKg,
			maximumValidWeightKg,
		)
	}

	return roundedWeightKg, nil
}

// NormalizeHeightToCentimeters converts a height measurement in "cm" or "in" into centimeters rounded to 2 decimal places,
// rejecting non-positive inputs or normalized values outside [80.0, 260.0] cm.
func NormalizeHeightToCentimeters(value float64, unit string) (float64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0, errors.New("height must be a positive number")
	}

	normalizedUnit := strings.ToLower(strings.TrimSpace(unit))
	if normalizedUnit == "" {
		normalizedUnit = "cm"
	}

	var heightCm float64
	switch normalizedUnit {
	case "cm", "cms", "centimeter", "centimeters":
		heightCm = value
	case "in", "inch", "inches":
		heightCm = value * centimetersPerInch
	default:
		return 0, fmt.Errorf("unsupported height unit %q (expected 'cm' or 'in')", unit)
	}

	roundedHeightCm := roundToTwoDecimalPlaces(heightCm)
	if roundedHeightCm < minimumValidHeightCm || roundedHeightCm > maximumValidHeightCm {
		return 0, fmt.Errorf(
			"normalized height %.2f cm is outside the valid range [%.1f, %.1f] cm",
			roundedHeightCm,
			minimumValidHeightCm,
			maximumValidHeightCm,
		)
	}

	return roundedHeightCm, nil
}

// CalculateAndValidateBMI computes BMI = round(weightKg / (heightCm/100)^2, 2),
// rejects non-positive inputs or unrealistic BMI values (< 10.0 or > 80.0),
// and returns the BMI classification ("underweight", "normal weight", "overweight", or "obese").
func CalculateAndValidateBMI(weightKg, heightCm float64) (bmi float64, classification string, err error) {
	if math.IsNaN(weightKg) || math.IsInf(weightKg, 0) || weightKg <= 0 {
		return 0, "", errors.New("weight in kilograms must be positive")
	}

	if math.IsNaN(heightCm) || math.IsInf(heightCm, 0) || heightCm <= 0 {
		return 0, "", errors.New("height in centimeters must be positive")
	}

	heightMeters := heightCm / 100.0
	rawBMI := weightKg / (heightMeters * heightMeters)
	roundedBMI := roundToTwoDecimalPlaces(rawBMI)

	if roundedBMI < minimumValidBMIValue || roundedBMI > maximumValidBMIValue {
		return 0, "", fmt.Errorf(
			"calculated BMI %.2f is outside the realistic human range [%.1f, %.1f]",
			roundedBMI,
			minimumValidBMIValue,
			maximumValidBMIValue,
		)
	}

	switch {
	case roundedBMI < underweightBMILimit:
		classification = "underweight"
	case roundedBMI < normalWeightBMILimit:
		classification = "normal weight"
	case roundedBMI < overweightBMILimit:
		classification = "overweight"
	default:
		classification = "obese"
	}

	return roundedBMI, classification, nil
}
