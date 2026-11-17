#include <iomanip>
#include <iostream>
#include <stdexcept>
#include <string>

constexpr double cmThreshold = 3;
constexpr double underweightMax = 18.5;
constexpr double healthyMax = 25;
constexpr double overweightMax = 30;

double toMeters(double h){
    return h > cmThreshold ? h /100 : h;
}

void validate(double w, double h) {
    if (w <= 0) {
        throw std::invalid_argument("weight must be greater than 0");
    }
    if (h <= 0) {
        throw std::invalid_argument("height must be greater than 0");
    }
}

double getBmi(double w, double h) {
    double m = toMeters(h);
    validate(w, m);
    return w / (m * m);
}

std::string sortBmi(double b) {
    if (b < underweightMax) return "underweight";
    if (b < healthyMax) return "healthy";
    if (b < overweightMax) return "overweight";
    return "obesity";
}

int main() {
    double height = 173;
    double weight = 64.3;

    try {
        double bmi = getBmi(weight, height);
        std::cout << std::fixed << std::setprecision(2)
            << "Your BMI: " << bmi
            << ", and you are "  << sortBmi(bmi) << "\n";
    } catch (const std::invalid_argument& e) {
        std::cout << e.what() << "\n";
        return 1;
    }
    return 0;
}