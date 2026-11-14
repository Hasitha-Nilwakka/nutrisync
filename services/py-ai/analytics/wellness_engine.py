import numpy as np

def calculate_bmi_and_classification(weight_kg: float, height_cm: float) -> tuple[float, str]:
    bmi = np.round((weight_kg / (height_cm/100) * (height_cm/100)),2)
    bmi_clipped = np.clip(bmi, 0, 100)

    conditions = [
        bmi_clipped < 18.5,
        (bmi_clipped >= 18.5) & (bmi_clipped < 25),
        (bmi_clipped >= 25) & (bmi_clipped < 30)
    ]

    choices = [
        "underweight",
        "normal weight",
        "overweight"
    ]

    return [bmi_clipped, np.select(conditions, choices, default="obese")]

def calculate_bmi_component_score(bmi: float) -> float:
    


