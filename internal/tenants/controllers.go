package tenants

import (
	"uuid"

	"github.com/alexedwards/scs/v2"
	"github.com/duddy57/toaki-server/internal/shared"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
)

type OrganizationResources struct {
	OrganizationService Service
	Session             *scs.SessionManager
}

func (rs OrganizationResources) Routes(s *fuego.Server) {
	organizationGroup := fuego.Group(s, "/organizations", option.Middleware(shared.AuthMiddleware(rs.Session)))

	fuego.Post(
		organizationGroup,
		"/create",
		rs.postCreateOrganization,
		option.Summary("Create organization"),
		option.OperationID("createOrganization"),
		option.Description("Operation for create a organization in system"),
	)
	fuego.Get(
		organizationGroup,
		"/{organizationID}",
		rs.getOrganizationDetails,
		option.Summary("Retrieve organization"),
		option.OperationID("getOrganizationDetails"),
		option.Description("Operation for retrieve all organization details"),
	)
	fuego.Put(organizationGroup,
		"/{organizationID}",
		rs.putOrganization,
		option.Summary("Update Organization"),
		option.OperationID("updateOrganization"),
		option.Description("Operation for alter details from organization"),
	)
	fuego.Delete(organizationGroup,
		"/{organizationID}",
		rs.deleteOrganizations,
		option.Summary("Delete Organization"),
		option.OperationID("deleteOrganization"),
		option.Description("Operation for delete organization"),
	)

}

func (rs OrganizationResources) postCreateOrganization(c fuego.ContextWithBody[CreateOrganization]) (*CreateOrganizaationSuccess, error) {
	body, err := c.Body()
	if err != nil {
		return &CreateOrganizaationSuccess{}, err
	}

	id, err := rs.OrganizationService.CreateOrganization(c.Context(), body)
	if err != nil {
		return &CreateOrganizaationSuccess{}, err
	}

	return &CreateOrganizaationSuccess{
		ID:      id,
		Message: "Organização criada com sucesso!",
	}, nil

}
func (rs OrganizationResources) getOrganizationDetails(c fuego.ContextNoBody) (Organizations, error) {
	organizationIdStr := c.PathParam("organizationID")
	organizationID := uuid.MustParse(organizationIdStr)

	organization, err := rs.OrganizationService.GetOrganization(c.Context(), organizationID)
	if err != nil {
		return Organizations{}, err
	}

	return organization, nil
}
func (rs OrganizationResources) deleteOrganizations(c fuego.ContextNoBody) (shared.SuccessResponse, error) {
	organizationIdStr := c.PathParam("organizationID")
	organizationID := uuid.MustParse(organizationIdStr)

	if err := rs.OrganizationService.DeleteOrganization(c.Context(), organizationID); err != nil {
		return shared.SuccessResponse{}, err
	}

	return shared.SuccessResponse{
		Message: "Organization deleted succesfully",
	}, nil
}
func (rs OrganizationResources) putOrganization(c fuego.ContextWithBody[UpdateRequest]) (shared.SuccessResponse, error) {
	body, err := c.Body()
	if err != nil {
		return shared.SuccessResponse{}, err
	}

	organizationIdStr := c.PathParam("organizationID")
	organizationID := uuid.MustParse(organizationIdStr)

	if err := rs.OrganizationService.UpdateOrganization(c.Context(), body, organizationID); err != nil {
		return shared.SuccessResponse{}, err
	}

	return shared.SuccessResponse{
		Message: "Organização atualizado com sucesso!",
	}, nil
}
