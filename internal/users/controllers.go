package users

import (
	"github.com/alexedwards/scs/v2"
	"github.com/duddy57/toaki-server/internal/shared"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
)

type UsersResources struct {
	UsersService Service
	Session      *scs.SessionManager
}

func (rs UsersResources) Routes(s *fuego.Server) {
	usersGroup := fuego.Group(s, "/users")

	fuego.Post(
		usersGroup,
		"/create",
		rs.postUsers,
		option.Summary("Create user"),
		option.OperationID("createUser"),
		option.Description("Operation for create a user on system"),
	)
	fuego.Post(
		usersGroup,
		"/login",
		rs.postLogin,
		option.Summary("Login user"),
		option.OperationID("loginUser"),
		option.Description("Operation for login a user on system"),
	)
	fuego.Post(
		usersGroup,
		"/request-password",
		rs.postRequestUpdatePassword,
		option.Summary("Lost password"),
		option.OperationID("requestPasswordUpdate"),
		option.Description("Operation for request the email to alter the user password"),
	)
	fuego.Patch(
		usersGroup,
		"/update-password/{resetToken}",
		rs.patchPassword,
		option.Summary("Alter password"),
		option.OperationID("updatePassword"),
		option.Description("Operation for alter the user password"),
	)

	protectedUserGroup := fuego.Group(usersGroup, "/", option.Middleware(shared.AuthMiddleware(rs.Session)))
	fuego.Get(
		protectedUserGroup,
		"/logout",
		rs.getLogout,
		option.Summary("Logout"),
		option.OperationID("logoutUser"),
		option.Description("Operation for logout"),
	)
	fuego.Get(
		protectedUserGroup,
		"/details",
		rs.getUser,
		option.Summary("User details"),
		option.OperationID("getUser"),
		option.Description("Operation for get user details"),
	)
	fuego.Delete(
		protectedUserGroup,
		"/delete",
		rs.deleteUser,
		option.Summary("Delete"),
		option.OperationID("deleteUser"),
		option.Description("Operation for delete user for system(Soft delete)"),
	)
	fuego.Put(
		protectedUserGroup,
		"/update",
		rs.putUsers,
		option.Summary("Update"),
		option.OperationID("updateUser"),
		option.Description("Operation for update user details"),
	)

}

func (rs UsersResources) postUsers(c fuego.ContextWithBody[CreateUserRequest]) (*CreateUserSuccess, error) {
	body, err := c.Body()
	if err != nil {
		return &CreateUserSuccess{}, err
	}

	id, err := rs.UsersService.CreateUsers(c.Context(), body)
	if err != nil {
		return &CreateUserSuccess{}, err
	}

	return &CreateUserSuccess{
		ID:      id,
		Message: "Seja bem vindo!",
	}, nil

}
func (rs UsersResources) postLogin(c fuego.ContextWithBody[LoginRequest]) (shared.SuccessResponse, error) {
	body, err := c.Body()
	if err != nil {
		return shared.SuccessResponse{}, err
	}

	if err := rs.UsersService.LoginUsers(c.Context(), body); err != nil {
		return shared.SuccessResponse{}, err
	}

	return shared.SuccessResponse{
		Message: "Bem vindo!",
	}, nil
}
func (rs UsersResources) postRequestUpdatePassword(c fuego.ContextWithBody[RequestUpdatePassword]) (shared.SuccessResponse, error) {
	body, err := c.Body()
	if err != nil {
		return shared.SuccessResponse{}, err
	}

	if err := rs.UsersService.RequestPasswordReset(c.Context(), body); err != nil {
		return shared.SuccessResponse{}, err
	}

	return shared.SuccessResponse{
		Message: "Verifique seu email ou spam!",
	}, nil
}
func (rs UsersResources) patchPassword(c fuego.ContextWithBody[UpdatePasswordRequest]) (shared.SuccessResponse, error) {
	body, err := c.Body()
	if err != nil {
		return shared.SuccessResponse{}, err
	}

	resetToken := c.PathParam("resetToken")

	if err := rs.UsersService.ResetPassword(c.Context(), resetToken, body); err != nil {
		return shared.SuccessResponse{}, err
	}

	return shared.SuccessResponse{
		Message: "Senha alterada com sucesso",
	}, nil
}
func (rs UsersResources) getLogout(c fuego.ContextNoBody) (shared.SuccessResponse, error) {
	if err := rs.UsersService.LogoutUsers(c.Context()); err != nil {
		return shared.SuccessResponse{}, err
	}

	return shared.SuccessResponse{
		Message: "Até mais",
	}, nil
}
func (rs UsersResources) getUser(c fuego.ContextNoBody) (User, error) {
	user, err := rs.UsersService.GetUser(c.Context())
	if err != nil {
		return User{}, err
	}

	return user, err
}
func (rs UsersResources) deleteUser(c fuego.ContextNoBody) (shared.SuccessResponse, error) {
	if err := rs.UsersService.DeleteUsers(c.Context()); err != nil {
		return shared.SuccessResponse{}, err
	}

	return shared.SuccessResponse{
		Message: "User deleted succesfully",
	}, nil
}
func (rs UsersResources) putUsers(c fuego.ContextWithBody[UpdateRequest]) (shared.SuccessResponse, error) {
	body, err := c.Body()
	if err != nil {
		return shared.SuccessResponse{}, err
	}

	if err := rs.UsersService.UpdateUsers(c.Context(), body); err != nil {
		return shared.SuccessResponse{}, err
	}

	return shared.SuccessResponse{
		Message: "Perfil atualizado com sucesso!",
	}, nil
}
