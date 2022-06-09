package main

type ParamType struct {
	Name string
	Type string
}

type Factor struct {
	FactorName  string
	FactorCode  string
	Description string
	ParamTypes  []ParamType
}
