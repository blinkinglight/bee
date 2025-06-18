package main

type UserCreated struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

type UserUpdated struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

type UserNameChanged struct {
	Name string `json:"name"`
}
type UserDeleted struct {
}
