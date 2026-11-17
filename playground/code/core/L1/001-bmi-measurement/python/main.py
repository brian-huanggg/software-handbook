"""A measurement for BMI"""

def to_meters(height: float) -> float:
    """Convert height to meters"""
    return height / 100 if height > 3 else height
def validate(weight: float, height: float) -> None:
    if weight <= 0:
        raise ValueError("weight must be > 0")
    if height <= 0:
        raise ValueError("height must be > 0")


def get_bmi(weight: float, height: float) -> float:
    """weight: kg, height: m or cm"""
    height = to_meters(height)
    validate(weight, height)
    return weight / pow(height, 2)

def sort_bmi(bmi: float) -> str:
    """Sort BMI"""
    if bmi < 18.5:
        return "Underweight"
    elif bmi < 25:
        return "Healthy"
    elif bmi < 30:
        return "Overweight"
    else:
        return "Obesity"

def main():
    try: 
        my_BMI = get_bmi(92, 1.7)
    except ValueError as e:
        print(f"Invalid input: {e}")
        return

    result = sort_bmi(my_BMI)
    print(f"Your BMI is {my_BMI:.2f}, you are {result}")

if __name__ == "__main__":
    main()