# Pseudocode

```
CONSTANT CM_THRESHOLD = 3          // 大於此值視為公分
CONSTANT UNDERWEIGHT_MAX = 18.5
CONSTANT HEALTHY_MAX = 24
CONSTANT OVERWEIGHT_MAX = 30

FUNCTION to_meters(height: float) -> float
    IF height > CM_THRESHOLD THEN
        RETURN height / 100
    END IF
    RETURN height
END FUNCTION

FUNCTION validate(weight: float, height: float)
    IF weight <= 0 THEN
        RAISE error("weight must be > 0")
    END IF
    IF height <= 0 THEN
        RAISE error("height must be > 0")
    END IF
END FUNCTION

FUNCTION get_bmi(weight: float, height: float) -> float
    height = to_meters(height)
    validate(weight, height)
    RETURN weight / (height * height)      // 不在這裡 round
END FUNCTION

FUNCTION sort_bmi(bmi: float) -> string
    IF bmi < UNDERWEIGHT_MAX THEN RETURN "Underweight"
    ELSE IF bmi < HEALTHY_MAX THEN RETURN "Healthy"
    ELSE IF bmi < OVERWEIGHT_MAX THEN RETURN "Overweight"
    ELSE RETURN "Obesity"
    END IF
END FUNCTION

FUNCTION main()
    weight = 62
    height = 1.7

    TRY
        bmi = get_bmi(weight, height)
    CATCH error AS e
        PRINT "Invalid input: " + e.message
        RETURN
    END TRY

    category = sort_bmi(bmi)
    PRINT "Your BMI is " + format(bmi, 2 decimals) + ", you are " + category
END FUNCTION

CALL main()      // 程式入口
```
