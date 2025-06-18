package main

type CreateUserCommand struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

type UpdateUserCommand struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}

type ChangeUserNameCommand struct {
	Name string `json:"name"`
}

type DeleteUserCommand struct {
}
