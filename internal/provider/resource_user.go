package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &userResource{}
var _ resource.ResourceWithConfigure = &userResource{}
var _ resource.ResourceWithImportState = &userResource{}

func init() {
	registeredResources = append(registeredResources, NewUserResource)
}

// userResource manages Open WebUI user accounts.
type userResource struct {
	client *client.Client
}

// userResourceModel captures Terraform state for a user account. Password is
// write-only: Terraform sends it to the provider from the configuration and
// never records it, so it is null in every state this resource writes.
type userResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Email           types.String `tfsdk:"email"`
	Role            types.String `tfsdk:"role"`
	ProfileImageURL types.String `tfsdk:"profile_image_url"`
	Password        types.String `tfsdk:"password"`
	PasswordVersion types.String `tfsdk:"password_version"`
	Username        types.String `tfsdk:"username"`
	LastActiveAt    types.Int64  `tfsdk:"last_active_at"`
	CreatedAt       types.Int64  `tfsdk:"created_at"`
	UpdatedAt       types.Int64  `tfsdk:"updated_at"`
}

// NewUserResource constructs a new user resource.
func NewUserResource() resource.Resource {
	return &userResource{}
}

// Metadata sets the resource type name.
func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the user resource schema.
func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a user account in Open WebUI.\n\n" +
			"Group membership belongs to `openwebui_group`, which owns the `users` list. There is no route that sets a user's groups from the user's side.\n\n" +
			"Open WebUI protects the first account ever created, its primary admin. Nobody else can update it, it cannot give up its own admin role, and no one can delete it. " +
			"This resource checks all three before it calls the API, so a plan that would demote or delete the primary admin fails instead of sending the request.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "UUID assigned by Open WebUI on create.",
				MarkdownDescription: "UUID assigned by Open WebUI on create.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Display name of the account.",
				MarkdownDescription: "Display name of the account.",
			},
			"email": schema.StringAttribute{
				Required:            true,
				Description:         "Mail address the account signs in with. Must be lowercase, because Open WebUI lowercases the address it stores.",
				MarkdownDescription: "Mail address the account signs in with. Must be lowercase, because Open WebUI lowercases the address it stores.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					lowercaseValidator{},
				},
			},
			"role": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Role of the account: `pending`, `user`, or `admin`. A new account is `pending` unless this says otherwise.",
				MarkdownDescription: "Role of the account: `pending`, `user`, or `admin`. A new account is `pending` unless this says otherwise.",
				Validators: []validator.String{
					stringvalidator.OneOf("pending", "user", "admin"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"profile_image_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Profile image of the account, as a path or a data URI.",
				MarkdownDescription: "Profile image of the account, as a path or a data URI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
				Description: "Password of the account. Required to create one. Terraform sends this value to the provider and never writes it to state, " +
					"and no Open WebUI route reads a password back, so a password changed elsewhere is invisible here. Change password_version to send a new one.",
				MarkdownDescription: "Password of the account. Required to create one. Terraform sends this value to the provider and never writes it to state, " +
					"and no Open WebUI route reads a password back, so a password changed elsewhere is invisible here. Change `password_version` to send a new one.",
			},
			"password_version": schema.StringAttribute{
				Optional: true,
				Description: "Marker that triggers a password write. The provider sends the password only while creating the account and whenever this value changes, " +
					"so an unrelated update never resets the password. Any text works, e.g. a date or a counter.",
				MarkdownDescription: "Marker that triggers a password write. The provider sends the password only while creating the account and whenever this value changes, " +
					"so an unrelated update never resets the password. Any text works, e.g. a date or a counter.",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				Description:         "Username of the account. Set by Open WebUI.",
				MarkdownDescription: "Username of the account. Set by Open WebUI.",
			},
			"last_active_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of the account's last activity. Set by Open WebUI.",
				MarkdownDescription: "Unix timestamp of the account's last activity. Set by Open WebUI.",
			},
			"created_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of when the account was created. Set by Open WebUI.",
				MarkdownDescription: "Unix timestamp of when the account was created. Set by Open WebUI.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of the account's last change. Set by Open WebUI.",
				MarkdownDescription: "Unix timestamp of the account's last change. Set by Open WebUI.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create provisions a user account.
func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing users.")
		return
	}

	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A write-only attribute is null in the plan by design. Its value arrives
	// with the configuration instead.
	var config userResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Password.IsNull() || config.Password.IsUnknown() || config.Password.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Password required",
			"Open WebUI needs a password to create an account. Set password, and set password_version alongside it so later rotations are visible in the plan.",
		)
		return
	}

	form := client.AddUserForm{
		Name:     plan.Name.ValueString(),
		Email:    plan.Email.ValueString(),
		Password: config.Password.ValueString(),
	}
	if !plan.Role.IsNull() && !plan.Role.IsUnknown() {
		role := plan.Role.ValueString()
		form.Role = &role
	}
	if !plan.ProfileImageURL.IsNull() && !plan.ProfileImageURL.IsUnknown() {
		image := plan.ProfileImageURL.ValueString()
		form.ProfileImageURL = &image
	}

	created, err := r.client.AddUser(ctx, form)
	if err != nil {
		resp.Diagnostics.AddError("Create user failed", userCreateErrorDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, userToModel(created, plan.PasswordVersion))...)
}

// Read refreshes user state.
func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing users.")
		return
	}

	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetUser(ctx, state.ID.ValueString())
	if err != nil {
		if userIsGone(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read user failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, userToModel(user, state.PasswordVersion))...)
}

// Update changes a user account. Only the attributes that differ from state go
// on the wire, and the password goes only when password_version changes.
func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing users.")
		return
	}

	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config userResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := state.ID.ValueString()

	form := client.UserUpdateForm{}
	if !plan.Name.Equal(state.Name) {
		name := plan.Name.ValueString()
		form.Name = &name
	}
	if !plan.Email.Equal(state.Email) {
		email := plan.Email.ValueString()
		form.Email = &email
	}
	if !plan.Role.IsUnknown() && !plan.Role.Equal(state.Role) {
		role := plan.Role.ValueString()
		form.Role = &role
	}
	if !plan.ProfileImageURL.IsUnknown() && !plan.ProfileImageURL.Equal(state.ProfileImageURL) {
		image := plan.ProfileImageURL.ValueString()
		form.ProfileImageURL = &image
	}

	if !plan.PasswordVersion.Equal(state.PasswordVersion) {
		if config.Password.IsNull() || config.Password.IsUnknown() || config.Password.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("password"),
				"Password required",
				"password_version changed, which asks the provider to write a new password, but password holds no value.",
			)
			return
		}
		password := config.Password.ValueString()
		form.Password = &password
	}

	resp.Diagnostics.Append(r.guardPrimaryAdminUpdate(ctx, userID, form.Role)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateUser(ctx, userID, form)
	if err != nil {
		resp.Diagnostics.AddError("Update user failed", userWriteErrorDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, userToModel(updated, plan.PasswordVersion))...)
}

// Delete removes a user account.
func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing users.")
		return
	}

	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := state.ID.ValueString()

	resp.Diagnostics.Append(r.guardPrimaryAdminDelete(ctx, userID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteUser(ctx, userID); err != nil {
		if userIsGone(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Delete user failed", userWriteErrorDetail(err))
		return
	}
}

// ImportState maps an import identifier onto the id attribute. The password is
// not readable, so an imported account carries none until one is written.
func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// guardPrimaryAdminUpdate refuses an update Open WebUI protects the primary
// admin against: a role that is not admin, and any change made by an account
// that is not the primary admin itself. The check runs before the request, so
// the refusal costs nothing and reads as a plan-time mistake rather than a 403.
//
// The guard fails closed. If the identity of the primary admin cannot be
// established, the update does not happen.
func (r *userResource) guardPrimaryAdminUpdate(ctx context.Context, userID string, plannedRole *string) diag.Diagnostics {
	var diags diag.Diagnostics

	primary, session, lookupDiags := r.loadProtectedIdentities(ctx)
	diags.Append(lookupDiags...)
	if diags.HasError() {
		return diags
	}

	if primary == nil || primary.ID != userID {
		return diags
	}

	if plannedRole != nil && *plannedRole != "admin" {
		diags.AddAttributeError(
			path.Root("role"),
			"Refusing to demote the primary admin",
			fmt.Sprintf("%s is the first account this Open WebUI instance created, and Open WebUI refuses to take the admin role away from it. Leave role as admin.", primary.Email),
		)
	}

	if session == nil || session.ID != primary.ID {
		diags.AddError(
			"Refusing to update the primary admin",
			fmt.Sprintf("%s is the first account this Open WebUI instance created, and only that account may change it. The provider's token belongs to a different account.", primary.Email),
		)
	}

	return diags
}

// guardPrimaryAdminDelete refuses to delete the primary admin, and refuses to
// delete the account whose token the provider uses. Open WebUI answers 403 to
// both, so the request is never worth making.
//
// The guard fails closed. If the identity of the primary admin cannot be
// established, the delete does not happen.
func (r *userResource) guardPrimaryAdminDelete(ctx context.Context, userID string) diag.Diagnostics {
	var diags diag.Diagnostics

	primary, session, lookupDiags := r.loadProtectedIdentities(ctx)
	diags.Append(lookupDiags...)
	if diags.HasError() {
		return diags
	}

	if primary != nil && primary.ID == userID {
		diags.AddError(
			"Refusing to delete the primary admin",
			fmt.Sprintf("%s is the first account this Open WebUI instance created, and Open WebUI refuses to delete it. Remove the resource from state with `terraform state rm` if it should no longer be managed here.", primary.Email),
		)
	}

	if session != nil && session.ID == userID {
		diags.AddError(
			"Refusing to delete the provider's own account",
			fmt.Sprintf("%s is the account the provider's token belongs to, and Open WebUI refuses to let an account delete itself.", session.Email),
		)
	}

	return diags
}

// loadProtectedIdentities reports which account is the primary admin and which
// account the provider authenticates as.
func (r *userResource) loadProtectedIdentities(ctx context.Context) (*client.User, *client.SessionUser, diag.Diagnostics) {
	var diags diag.Diagnostics

	primary, err := r.client.GetPrimaryAdmin(ctx)
	if err != nil {
		diags.AddError(
			"Unable to identify the primary admin",
			fmt.Sprintf("Open WebUI protects the account it created first, and the provider refuses to write to an account it cannot rule out: %v", err),
		)
		return nil, nil, diags
	}

	session, err := r.client.GetSessionUser(ctx)
	if err != nil {
		diags.AddError(
			"Unable to identify the provider's own account",
			fmt.Sprintf("The primary admin check needs to know which account the token belongs to: %v", err),
		)
		return nil, nil, diags
	}

	return primary, session, diags
}

// userIsGone reports whether an error says the account no longer exists. The
// read route answers 400 rather than 404 when it finds nothing, and a lookup by
// ID has no other way to fail with either status.
func userIsGone(err error) bool {
	if errors.Is(err, client.ErrNotFound) {
		return true
	}

	var apiErr *client.APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusBadRequest
}

// userCreateErrorDetail explains the two ways a create fails. Open WebUI
// answers 400 for an address that already has an account and for a password its
// own rules reject, and there is no upsert to fall back on.
func userCreateErrorDetail(err error) string {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) && apiErr.Status == http.StatusBadRequest {
		return fmt.Sprintf("%v\n\nOpen WebUI refuses an address that already has an account, and refuses a password that fails its own rules. To manage an account that already exists, import it instead of creating it.", err)
	}

	return err.Error()
}

// userWriteErrorDetail explains a refusal Open WebUI answers with 403. The
// guards catch these before the request, so one arriving here means the
// instance changed under the apply.
func userWriteErrorDetail(err error) string {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) && apiErr.Status == http.StatusForbidden {
		return fmt.Sprintf("%v\n\nOpen WebUI refuses this action on its primary admin, and refuses to let an account delete itself.", err)
	}

	return err.Error()
}

// userToModel converts an API user into Terraform state. The password stays
// null: it is write-only, and Terraform rejects a state that holds one.
func userToModel(user *client.User, passwordVersion types.String) userResourceModel {
	username := types.StringNull()
	if user.Username != nil {
		username = types.StringValue(*user.Username)
	}

	return userResourceModel{
		ID:              types.StringValue(user.ID),
		Name:            types.StringValue(user.Name),
		Email:           types.StringValue(user.Email),
		Role:            types.StringValue(user.Role),
		ProfileImageURL: types.StringValue(user.ProfileImage),
		Password:        types.StringNull(),
		PasswordVersion: passwordVersion,
		Username:        username,
		LastActiveAt:    types.Int64Value(user.LastActiveAt),
		CreatedAt:       types.Int64Value(user.CreatedAt),
		UpdatedAt:       types.Int64Value(user.UpdatedAt),
	}
}
