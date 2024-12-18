package main

import (
	"fmt"
	"math"
)

// math.Asin()
func GOTOSH_FUNC_math_Asin(x float64) float64 {
	if x == 1 {
		return math.Pi / 2
	} else if x == -1 {
		return -math.Pi / 2
	}
	return math.Atan(x / math.Sqrt(1-x*x))
}

// math.Acos()
func GOTOSH_FUNC_math_Acos(x float64) float64 {
	if x == 0 {
		return math.Pi / 2
	} else if x == 1 {
		return 0
	} else if x == -1 {
		return math.Pi
	}
	if x > 0 {
		return math.Atan(math.Sqrt(1-x*x) / x)
	} else {
		return math.Pi + math.Atan(math.Sqrt(1-x*x)/x)
	}
}

type Vector2 struct {
	X float64
	Y float64
}

func (v Vector2) Len() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

func (v Vector2) Dot(v2 Vector2) float64 {
	return v.X*v2.X + v.Y*v2.Y
}

func (v Vector2) Normalized() Vector2 {
	l := v.Len()
	return Vector2{v.X / l, v.Y / l}
}

func main() {
	// float
	var a = 1.5 * 1.5
	fmt.Printf("%.10f\n", a*2.5*3)
	fmt.Println(int(a))

	fmt.Printf("%.10f\n", math.Pi)
	fmt.Printf("%.10f\n", math.E)
	fmt.Printf("%.10f\n", math.Sqrt(2))
	fmt.Printf("%.10f\n", math.Exp(2))
	fmt.Printf("%.10f\n", math.Log(2))
	fmt.Printf("%.10f\n", math.Pow(2, 10))
	fmt.Printf("%.10f\n", math.Sin(2))
	fmt.Printf("%.10f\n", math.Cos(2))
	fmt.Printf("%.10f\n", math.Tan(2))
	fmt.Printf("%.10f\n", math.Atan(2))
	fmt.Printf("%.10f\n", math.Sinh(2))
	fmt.Printf("%.10f\n", math.Cosh(2))
	fmt.Printf("%.10f\n", math.Tanh(2))

	fmt.Printf("Asin %.10f\n", math.Asin(-0.5))
	fmt.Printf("Acos %.10f\n", math.Acos(-0.5))

	v := Vector2{1.5, 1.5}
	fmt.Printf("Len(): %.10f\n", v.Len())
	fmt.Printf("Dot(): %.10f\n", v.Dot(Vector2{1.5, -1.5}))
	n := v.Normalized()
	fmt.Printf("Normal: %.10f, %.10f\n", n.X, n.Y)
}
