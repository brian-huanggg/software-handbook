const CM_THRESHOLD = 3 // Recognize meter if height exceeds the threshold
const UNDERWEIGHT_MAX = 18.5
const HEALTHY_MAX = 24
const OVERWEIGHT_MAX = 30

function toMeters(height: number): number {
    return height > CM_THRESHOLD ? height / 100 : height
}

function validate(weight: number, height: number) {
    if (weight <= 0) {
        throw new Error("Weight must be greater than 0")
    }
    if (height <= 0) {
        throw new Error("Height must be greater than 0")
    }
}

function getBmi(weight: number, height: number): number {
    const meters = toMeters(height)
    validate(weight, meters)
    return weight / Math.pow(meters, 2)
}

function sortBmi(bmi: number): string {
    if (bmi < UNDERWEIGHT_MAX) {
        return "Underweight"
    } else if (bmi < HEALTHY_MAX) {
        return "Healthy"
    } else if (bmi < OVERWEIGHT_MAX) {
        return "Overweight"
    } else {
        return "Obesity"
    }
}

function main() {
    const weight = 62
    const height = 1.7

    try {
        const bmi = getBmi(weight, height)
        const category = sortBmi(bmi)
        console.log(`Your bmi is ${bmi.toFixed(2)}, and you are ${category}`)
    } catch (e) {
        console.log(`Invalid Input: ${e instanceof Error ? e.message : e}`)
    }
}

main()