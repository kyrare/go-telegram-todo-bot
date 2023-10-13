package main

type ValidationError struct {
	text string
	Err  error
}

func (v *ValidationError) Error() string {
	return v.text
}
