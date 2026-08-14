package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &ldapConfigResource{}
var _ resource.ResourceWithConfigure = &ldapConfigResource{}
var _ resource.ResourceWithImportState = &ldapConfigResource{}
var _ resource.ResourceWithValidateConfig = &ldapConfigResource{}

func init() {
	registeredResources = append(registeredResources, NewLDAPConfigResource)
}

// ldapConfigResource manages the LDAP directory connection and the switch that
// turns LDAP sign-in on. Open WebUI splits the two across separate routes; the
// admin panel presents them together, and so does this resource.
type ldapConfigResource struct {
	client *client.Client
}

type ldapConfigModel struct {
	ID                    types.String `tfsdk:"id"`
	EnableLDAP            types.Bool   `tfsdk:"enable_ldap"`
	Label                 types.String `tfsdk:"label"`
	Host                  types.String `tfsdk:"host"`
	Port                  types.Int64  `tfsdk:"port"`
	AttributeForMail      types.String `tfsdk:"attribute_for_mail"`
	AttributeForUsername  types.String `tfsdk:"attribute_for_username"`
	AppDN                 types.String `tfsdk:"app_dn"`
	AppDNPassword         types.String `tfsdk:"app_dn_password"`
	SearchBase            types.String `tfsdk:"search_base"`
	SearchFilters         types.String `tfsdk:"search_filters"`
	UseTLS                types.Bool   `tfsdk:"use_tls"`
	CertificatePath       types.String `tfsdk:"certificate_path"`
	ValidateCert          types.Bool   `tfsdk:"validate_cert"`
	Ciphers               types.String `tfsdk:"ciphers"`
	EnableGroupManagement types.Bool   `tfsdk:"enable_group_management"`
	EnableGroupCreation   types.Bool   `tfsdk:"enable_group_creation"`
	AttributeForGroups    types.String `tfsdk:"attribute_for_groups"`
}

// NewLDAPConfigResource constructs a new LDAP config resource.
func NewLDAPConfigResource() resource.Resource {
	return &ldapConfigResource{}
}

// Metadata sets the resource type name.
func (r *ldapConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ldap_config"
}

// Schema defines the LDAP config schema.
func (r *ldapConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the LDAP directory Open WebUI signs users in against, and the switch that enables LDAP sign-in.\n\n" +
			"Open WebUI stores the bind password in plain text and returns it on read, so it round-trips into Terraform state. Treat the state file as a secret.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier. Set by Open WebUI.",
				MarkdownDescription: "Singleton identifier. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_ldap": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether users can sign in through LDAP.",
				MarkdownDescription: "Whether users can sign in through LDAP.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"label": ldapStringAttribute("Name of the directory, shown on the sign-in page. Open WebUI rejects an empty value.", stringvalidator.LengthAtLeast(1)),
			"host":  ldapStringAttribute("Hostname or address of the LDAP server. Open WebUI rejects an empty value.", stringvalidator.LengthAtLeast(1)),
			"port": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Port of the LDAP server. Unset uses 389, or 636 with TLS.",
				MarkdownDescription: "Port of the LDAP server. Unset uses 389, or 636 with TLS.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"attribute_for_mail":     ldapStringAttribute("Directory attribute holding the user's mail address, e.g. `mail`. Open WebUI rejects an empty value.", stringvalidator.LengthAtLeast(1)),
			"attribute_for_username": ldapStringAttribute("Directory attribute holding the username, e.g. `uid`. Open WebUI rejects an empty value.", stringvalidator.LengthAtLeast(1)),
			"app_dn":                 ldapStringAttribute("Distinguished name Open WebUI binds with."),
			"app_dn_password": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Password for the bind DN. Open WebUI returns this value unmasked on read.",
				MarkdownDescription: "Password for the bind DN. Open WebUI returns this value unmasked on read.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"search_base":    ldapStringAttribute("Base DN the user search starts from. Open WebUI rejects an empty value.", stringvalidator.LengthAtLeast(1)),
			"search_filters": ldapStringAttribute("Extra LDAP filter applied to the user search."),
			"use_tls":        ldapBoolAttribute("Whether to connect over TLS."),
			"certificate_path": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Path to the CA certificate file used to verify the server.",
				MarkdownDescription: "Path to the CA certificate file used to verify the server.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"validate_cert": ldapBoolAttribute("Whether to verify the server's certificate."),
			"ciphers": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "TLS cipher list, e.g. `ALL`.",
				MarkdownDescription: "TLS cipher list, e.g. `ALL`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enable_group_management": ldapBoolAttribute("Whether Open WebUI reads group membership from the directory."),
			"enable_group_creation":   ldapBoolAttribute("Whether Open WebUI creates a group it finds in the directory but not locally."),
			"attribute_for_groups":    ldapStringAttribute("Directory attribute holding group membership, e.g. `memberOf`. Open WebUI rejects an empty value while `enable_group_management` is true."),
		},
	}
}

// ldapStringAttribute builds a text setting, with optional validators.
func ldapStringAttribute(description string, validators ...validator.String) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Description:         description,
		MarkdownDescription: description,
		Validators:          validators,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

// ldapBoolAttribute builds an on/off setting.
func ldapBoolAttribute(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Description:         description,
		MarkdownDescription: description,
		PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
	}
}

// ValidateConfig rejects the one combination Open WebUI refuses across two
// attributes: group management with no attribute to read groups from.
func (r *ldapConfigResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ldapConfigModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.EnableGroupManagement.IsNull() || config.EnableGroupManagement.IsUnknown() || !config.EnableGroupManagement.ValueBool() {
		return
	}

	if config.AttributeForGroups.IsNull() || config.AttributeForGroups.IsUnknown() {
		return
	}

	if config.AttributeForGroups.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("attribute_for_groups"),
			"Group attribute required",
			"Open WebUI refuses an empty attribute_for_groups while enable_group_management is true. Name the directory attribute that holds group membership.",
		)
	}
}

// Configure assigns the API client.
func (r *ldapConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create writes the LDAP config.
func (r *ldapConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing LDAP config.")
		return
	}

	var plan ldapConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyLDAPConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the LDAP config.
func (r *ldapConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing LDAP config.")
		return
	}

	server, err := r.client.GetLDAPServerConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read LDAP server config failed", err.Error())
		return
	}

	enabled, err := r.client.GetLDAPEnabled(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read LDAP config failed", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, ldapConfigToModel(server, enabled))...)
}

// Update writes the LDAP config.
func (r *ldapConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing LDAP config.")
		return
	}

	var plan ldapConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, diags := applyLDAPConfig(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the resource from state without changing remote configuration.
func (r *ldapConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing LDAP config.")
		return
	}
}

// ImportState maps import identifiers onto the id attribute.
func (r *ldapConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyLDAPConfig writes the directory connection, then the sign-in switch. The
// route takes all 16 directory fields at once and refuses a null for most of
// them, so the values the instance holds today fill in for the attributes the
// plan does not name. The switch goes last, so a connection that fails
// validation never leaves LDAP sign-in enabled against it.
func applyLDAPConfig(ctx context.Context, apiClient *client.Client, plan ldapConfigModel) (ldapConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	current, err := apiClient.GetLDAPServerConfig(ctx)
	if err != nil {
		diags.AddError("Read LDAP server config failed", err.Error())
		return ldapConfigModel{}, diags
	}

	config := *current
	overlayStringValue(&config.Label, plan.Label)
	overlayStringValue(&config.Host, plan.Host)
	overlayOptionalInt64Value(&config.Port, plan.Port)
	overlayStringValue(&config.AttributeForMail, plan.AttributeForMail)
	overlayStringValue(&config.AttributeForUsername, plan.AttributeForUsername)
	overlayStringValue(&config.AppDN, plan.AppDN)
	overlayStringValue(&config.AppDNPassword, plan.AppDNPassword)
	overlayStringValue(&config.SearchBase, plan.SearchBase)
	overlayStringValue(&config.SearchFilters, plan.SearchFilters)
	overlayBoolValue(&config.UseTLS, plan.UseTLS)
	overlayOptionalStringValue(&config.CertificatePath, plan.CertificatePath)
	overlayBoolValue(&config.ValidateCert, plan.ValidateCert)
	overlayOptionalStringValue(&config.Ciphers, plan.Ciphers)
	overlayBoolValue(&config.EnableGroupManagement, plan.EnableGroupManagement)
	overlayBoolValue(&config.EnableGroupCreation, plan.EnableGroupCreation)
	overlayStringValue(&config.AttributeForGroups, plan.AttributeForGroups)

	updated, err := apiClient.SetLDAPServerConfig(ctx, config)
	if err != nil {
		diags.AddError("Update LDAP server config failed", err.Error())
		return ldapConfigModel{}, diags
	}

	enabled, err := apiClient.GetLDAPEnabled(ctx)
	if err != nil {
		diags.AddError("Read LDAP config failed", err.Error())
		return ldapConfigModel{}, diags
	}

	if !plan.EnableLDAP.IsNull() && !plan.EnableLDAP.IsUnknown() && plan.EnableLDAP.ValueBool() != enabled {
		enabled, err = apiClient.SetLDAPEnabled(ctx, plan.EnableLDAP.ValueBool())
		if err != nil {
			diags.AddError("Update LDAP config failed", err.Error())
			return ldapConfigModel{}, diags
		}
	}

	return ldapConfigToModel(updated, enabled), diags
}

func ldapConfigToModel(config *client.LDAPServerConfig, enabled bool) ldapConfigModel {
	port := types.Int64Null()
	if config.Port != nil {
		port = types.Int64Value(*config.Port)
	}

	return ldapConfigModel{
		ID:                    types.StringValue("ldap"),
		EnableLDAP:            types.BoolValue(enabled),
		Label:                 types.StringValue(config.Label),
		Host:                  types.StringValue(config.Host),
		Port:                  port,
		AttributeForMail:      types.StringValue(config.AttributeForMail),
		AttributeForUsername:  types.StringValue(config.AttributeForUsername),
		AppDN:                 types.StringValue(config.AppDN),
		AppDNPassword:         types.StringValue(config.AppDNPassword),
		SearchBase:            types.StringValue(config.SearchBase),
		SearchFilters:         types.StringValue(config.SearchFilters),
		UseTLS:                types.BoolValue(config.UseTLS),
		CertificatePath:       stringValueOrNull(config.CertificatePath),
		ValidateCert:          types.BoolValue(config.ValidateCert),
		Ciphers:               stringValueOrNull(config.Ciphers),
		EnableGroupManagement: types.BoolValue(config.EnableGroupManagement),
		EnableGroupCreation:   types.BoolValue(config.EnableGroupCreation),
		AttributeForGroups:    types.StringValue(config.AttributeForGroups),
	}
}

// overlayOptionalInt64Value writes a planned value over a current one that the
// API allows to be null.
func overlayOptionalInt64Value(target **int64, planned types.Int64) {
	if planned.IsNull() || planned.IsUnknown() {
		return
	}
	value := planned.ValueInt64()
	*target = &value
}
