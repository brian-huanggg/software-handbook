public class Main {
    static final double CM_THRESHOLD = 3;
    static final double UNDERWEIGHT_MAX = 18.5;
    static final double HEALTHY_MAX = 25;
    static final double OVERWEIGHT_MAX = 30;

    static double toMeters(double h) {
        return h > CM_THRESHOLD ? h / 100 : h;
    }

    static void validate(double w, double h) {
        if (w <= 0) {
            throw new IllegalArgumentException("weight must be greater than 0");
        }
        if (h <= 0) {
            throw new IllegalArgumentException("height must be greater than 0");
        }
    }

    static double getBmi(double w, double h) {
        double m = toMeters(h);
        validate(w, m);
        return w / (m * m);
    }

    static String sortBmi(double b) {
        if (b < UNDERWEIGHT_MAX) return "underweight";
        if (b < HEALTHY_MAX) return "healthy";
        if (b < OVERWEIGHT_MAX) return "overweight";
        return "obesity";
    }

    public static void main(String[] args) {
        double weight = 64.3;
        double height = 173;
        
        try {
            double bmi = getBmi(weight, height);
            System.out.printf("Your BMI: %.2f, and you are %s%n", bmi, sortBmi(bmi));

        } catch (IllegalArgumentException e) {
            System.out.println(e.getMessage());
        }
    }
}