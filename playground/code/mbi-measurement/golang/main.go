package main

import (
	"errors"
	"fmt"
)

const (
	cmThreshold    = 3
	underweightMax = 18.5
	healthyMax     = 24
	overweightMax  = 30
)

func toMeters(h float64) float64 {
	if h > cmThreshold {
		return h / 100
	}
	return h
}

func validate(w, h float64) error {
	if w <= 0 {
		return errors.New("weight must be greater than 0")
	}
	if h <= 0 {
		return errors.New("height must be greater than 0")
	}
	return nil
}

func getBmi(w, h float64) (float64, error) {
	m := toMeters(h)
	if err := validate(w, m); err != nil {
		return 0, err
	}
	return w / (m * m), nil
}

func sortBmi(bmi float64) string {
	switch {
	case bmi < underweightMax:
		return "underweight"
	case bmi < healthyMax:
		return "healthy"
	case bmi < overweightMax:
		return "overweight"
	default:
		return "obesity"
	}
}

func main() {
	const weight = 64.3
	const height = 173

	bmi, err := getBmi(weight, height)
	if err != nil {
		fmt.Println(err)
		return
	}
	category := sortBmi(bmi)

	fmt.Printf("Your BMI: %.2f, and you are %s\n", bmi, category)
}
