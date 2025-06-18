package main

import "github.com/blinkinglight/bee"

func init() {
	bee.RegisterEvent[UserCreated]("users", "created")
	bee.RegisterEvent[UserUpdated]("users", "updated")
	bee.RegisterEvent[UserDeleted]("users", "deleted")
	bee.RegisterEvent[UserNameChanged]("users", "name_changed")

	bee.RegisterCommand[CreateUserCommand]("users", "create")
	bee.RegisterCommand[UpdateUserCommand]("users", "update")
	bee.RegisterCommand[ChangeUserNameCommand]("users", "change_name")
	bee.RegisterCommand[DeleteUserCommand]("users", "delete")
}
