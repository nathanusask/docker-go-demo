package main

type ParamType struct {
	Name string
	Type string
}

type Factor struct {
	FactorName  string
	Description string
	ParamTypes  []ParamType
}
