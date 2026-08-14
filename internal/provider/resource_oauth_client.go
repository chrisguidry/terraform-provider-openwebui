package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &oauthClientResource{}
var _ resource.ResourceWithConfigure = &oauthClientResource{}
var _ resource.ResourceWithImportState = &oauthClientResource{}
var _ resource.ResourceWithModifyPlan = &oauthClientResource{}

// oauthClientResource registers OAuth clients.
type oauthClientResource struct {
	client *client.Client
}

type oauthClientModel struct {
	ID             types.String `tfsdk:"id"`
	URL            types.String `tfsdk:"url"`
	ClientID       types.String `tfsdk:"client_id"`
	ClientName     types.String `tfsdk:"client_name"`
	ClientSecret   types.String `tfsdk:"client_secret"`
	OAuthServerURL types.String `tfsdk:"oauth_server_url"`
	OAuthScope     types.String `tfsdk:"oauth_scope"`
	Type           types.String `tfsdk:"type"`

	OAuthClientInfo types.String `tfsdk:"oauth_client_info"`
}

// NewOAuthClientResource constructs a new OAuth client resource.
func NewOAuthClientResource() resource.Resource {
	return &oauthClientResource{}
}

// Metadata sets the resource type name.
func (r *oauthClientResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_client"
}

// Schema defines the OAuth client schema.
func (r *oauthClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Registers an OAuth client with Open WebUI.\n\n~> **Note:** The OAuth client registration endpoint is write-only. Terraform preserves state from the last apply and cannot detect out-of-band changes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Mirrors `client_id`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "OAuth provider URL to register the client with. Forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"client_id": schema.StringAttribute{
				Required:    true,
				Description: "OAuth client identifier to register. Forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"client_name": schema.StringAttribute{
				Optional:    true,
				Description: "Optional display name for the OAuth client.",
			},
			"client_secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "OAuth client secret. Setting it makes Open WebUI build the registration from " +
					"these static credentials instead of running dynamic client registration, which is what an " +
					"`oauth_2.1_static` tool server needs. Sensitive.",
				MarkdownDescription: "OAuth client secret. Setting it makes Open WebUI build the registration " +
					"from these static credentials instead of running dynamic client registration, which is what " +
					"an `oauth_2.1_static` tool server needs. Sensitive.",
			},
			"oauth_server_url": schema.StringAttribute{
				Optional:            true,
				Description:         "Authorization server to register with, when it differs from `url`. Defaults to `url`.",
				MarkdownDescription: "Authorization server to register with, when it differs from `url`. Defaults to `url`.",
			},
			"oauth_scope": schema.StringAttribute{
				Optional:            true,
				Description:         "Space-separated OAuth scopes to request during registration. Open WebUI v0.11.0 and later accept this; older releases ignore it.",
				MarkdownDescription: "Space-separated OAuth scopes to request during registration. Open WebUI v0.11.0 and later accept this; older releases ignore it.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "OAuth client type, e.g. `mcp`. Open WebUI registers the client as `<type>:<client_id>`, which is the name a tool server of that type looks it up by. Forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"oauth_client_info": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
				Description: "Encrypted OAuth client registration returned by Open WebUI. Pass it to the " +
					"`oauth_client_info` attribute of an `openwebui_tool_server` resource, which is what makes " +
					"an `oauth_2.1` tool server authenticate. Open WebUI encrypts it with " +
					"`OAUTH_CLIENT_INFO_ENCRYPTION_KEY`, so Terraform carries the value and cannot read it. Sensitive.",
				MarkdownDescription: "Encrypted OAuth client registration returned by Open WebUI. Pass it to " +
					"the `oauth_client_info` attribute of an `openwebui_tool_server` resource, which is what " +
					"makes an `oauth_2.1` tool server authenticate. Open WebUI encrypts it with " +
					"`OAUTH_CLIENT_INFO_ENCRYPTION_KEY`, so Terraform carries the value and cannot read it. Sensitive.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *oauthClientResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create registers the OAuth client.
func (r *oauthClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OAuth clients.")
		return
	}

	var plan oauthClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyOAuthClientRegistration(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read preserves current state (no read endpoint available).
func (r *oauthClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state oauthClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update re-registers the OAuth client.
func (r *oauthClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OAuth clients.")
		return
	}

	var plan oauthClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyOAuthClientRegistration(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *oauthClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing OAuth clients.")
		return
	}
}

// ModifyPlan marks the registration blob unknown whenever anything else about
// the client changes. Registration runs again on every write and mints a new
// blob, so a plan that promised the stored one would not match the result.
func (r *oauthClientResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	if req.Plan.Raw.Equal(req.State.Raw) {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("oauth_client_info"), types.StringUnknown())...)
}

// ImportState maps import identifiers to client_id.
func (r *oauthClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("client_id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func applyOAuthClientRegistration(ctx context.Context, apiClient *client.Client, plan oauthClientModel) (oauthClientModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	var clientName *string
	if !plan.ClientName.IsNull() && !plan.ClientName.IsUnknown() {
		value := plan.ClientName.ValueString()
		clientName = &value
	}

	var clientSecret *string
	if !plan.ClientSecret.IsNull() && !plan.ClientSecret.IsUnknown() {
		value := plan.ClientSecret.ValueString()
		clientSecret = &value
	}

	var oauthServerURL *string
	if !plan.OAuthServerURL.IsNull() && !plan.OAuthServerURL.IsUnknown() {
		value := plan.OAuthServerURL.ValueString()
		oauthServerURL = &value
	}

	var oauthScope *string
	if !plan.OAuthScope.IsNull() && !plan.OAuthScope.IsUnknown() {
		value := plan.OAuthScope.ValueString()
		oauthScope = &value
	}

	form := client.OAuthClientRegistrationForm{
		URL:            plan.URL.ValueString(),
		ClientID:       plan.ClientID.ValueString(),
		ClientName:     clientName,
		ClientSecret:   clientSecret,
		OAuthServerURL: oauthServerURL,
		OAuthScope:     oauthScope,
	}

	var clientType *string
	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		value := plan.Type.ValueString()
		clientType = &value
	}

	resp, err := apiClient.RegisterOAuthClient(ctx, form, clientType)
	if err != nil {
		diags.AddError("Register OAuth client failed", err.Error())
		return oauthClientModel{}, diags
	}

	state := plan
	state.ID = types.StringValue(plan.ClientID.ValueString())
	state.OAuthClientInfo = types.StringNull()
	if info, ok := resp["oauth_client_info"].(string); ok {
		state.OAuthClientInfo = types.StringValue(info)
	}

	return state, diags
}
