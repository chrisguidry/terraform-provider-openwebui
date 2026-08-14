package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &terminalServerResource{}
var _ resource.ResourceWithConfigure = &terminalServerResource{}
var _ resource.ResourceWithImportState = &terminalServerResource{}

func init() {
	registeredResources = append(registeredResources, NewTerminalServerResource)
}

// terminalServerResource manages one entry of the TERMINAL_SERVER_CONNECTIONS
// list. Open WebUI stores the whole list under a single config key and replaces
// it on every write, so each change reads the list, splices this entry, and
// writes it back. The client serialises that cycle.
type terminalServerResource struct {
	client *client.Client
}

type terminalServerModel struct {
	ID         types.String `tfsdk:"id"`
	ServerID   types.String `tfsdk:"server_id"`
	Name       types.String `tfsdk:"name"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	URL        types.String `tfsdk:"url"`
	Path       types.String `tfsdk:"path"`
	Key        types.String `tfsdk:"key"`
	AuthType   types.String `tfsdk:"auth_type"`
	ConfigJSON types.String `tfsdk:"config_json"`
	ServerType types.String `tfsdk:"server_type"`
	PolicyID   types.String `tfsdk:"policy_id"`
}

// NewTerminalServerResource constructs a new terminal server resource.
func NewTerminalServerResource() resource.Resource {
	return &terminalServerResource{}
}

// Metadata sets the resource type name.
func (r *terminalServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_terminal_server"
}

// Schema defines the terminal server schema.
func (r *terminalServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages one terminal server connection in Open WebUI. A terminal server gives chats a shell, either directly or " +
			"through an orchestrator that hands out sessions.\n\n" +
			"Open WebUI keeps every terminal server in one list and replaces that list on each write, so the provider reads the list, " +
			"splices this entry, and writes it back. Connections it does not manage pass through untouched, apart from the `policy` and " +
			"`lifecycle` keys, which Open WebUI itself drops from every connection on every write. Two `terraform apply` runs against the " +
			"same Open WebUI at the same time can lose a connection: the lock that orders these writes lives in one provider process.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Terraform identifier. Always equal to `server_id`.",
				MarkdownDescription: "Terraform identifier. Always equal to `server_id`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"server_id": schema.StringAttribute{
				Required:            true,
				Description:         "Identifier of the connection within the terminal server list. Chosen by the practitioner, not by Open WebUI. Changing it replaces the connection.",
				MarkdownDescription: "Identifier of the connection within the terminal server list. Chosen by the practitioner, not by Open WebUI. Changing it replaces the connection.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Display name of the terminal server.",
				MarkdownDescription: "Display name of the terminal server.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether Open WebUI offers this terminal server. Defaults to `true`.",
				MarkdownDescription: "Whether Open WebUI offers this terminal server. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"url": schema.StringAttribute{
				Required:            true,
				Description:         "Base URL of the terminal server, e.g. `http://terminals.internal:8080`.",
				MarkdownDescription: "Base URL of the terminal server, e.g. `http://terminals.internal:8080`.",
			},
			"path": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Path to the terminal server's OpenAPI spec. Defaults to `/openapi.json`.",
				MarkdownDescription: "Path to the terminal server's OpenAPI spec. Defaults to `/openapi.json`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "API key or bearer token for the terminal server. Sensitive. Open WebUI returns it unmasked.",
				MarkdownDescription: "API key or bearer token for the terminal server. Sensitive. Open WebUI returns it unmasked.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"auth_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Authentication type for the terminal server, e.g. `bearer`. Defaults to `bearer`.",
				MarkdownDescription: "Authentication type for the terminal server, e.g. `bearer`. Defaults to `bearer`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"config_json": schema.StringAttribute{
				Optional:            true,
				Description:         "JSON object of extra configuration for the terminal server, e.g. `jsonencode({ timeout = 30 })`.",
				MarkdownDescription: "JSON object of extra configuration for the terminal server, e.g. `jsonencode({ timeout = 30 })`.",
			},
			"server_type": schema.StringAttribute{
				Optional:            true,
				Description:         "Kind of terminal server, `orchestrator` or `terminal`. Null lets Open WebUI detect it.",
				MarkdownDescription: "Kind of terminal server, `orchestrator` or `terminal`. Null lets Open WebUI detect it.",
			},
			"policy_id": schema.StringAttribute{
				Optional:            true,
				Description:         "Identifier of the policy the orchestrator applies to sessions from this connection. The policy itself lives on the orchestrator, not in Open WebUI.",
				MarkdownDescription: "Identifier of the policy the orchestrator applies to sessions from this connection. The policy itself lives on the orchestrator, not in Open WebUI.",
			},
		},
	}
}

// Configure assigns the API client.
func (r *terminalServerResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if apiClient, ok := req.ProviderData.(*client.Client); ok {
		r.client = apiClient
	}
}

// Create adds the connection to the terminal server list.
func (r *terminalServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing terminal servers.")
		return
	}

	var plan terminalServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn := expandTerminalServer(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	stored, err := r.client.AddTerminalServerConnection(ctx, conn)
	if err != nil {
		if errors.Is(err, client.ErrTerminalServerExists) {
			resp.Diagnostics.AddAttributeError(
				path.Root("server_id"),
				"Terminal server already registered",
				"Open WebUI already has a terminal server connection with the id "+conn.ID+". Choose another server_id, or import the existing connection with `terraform import`.",
			)
			return
		}
		resp.Diagnostics.AddError("Create terminal server failed", err.Error())
		return
	}

	state := flattenTerminalServer(stored, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes one connection from the terminal server list.
func (r *terminalServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing terminal servers.")
		return
	}

	var state terminalServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stored, err := r.client.GetTerminalServerConnection(ctx, state.ServerID.ValueString())
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read terminal server failed", err.Error())
		return
	}

	refreshed := flattenTerminalServer(stored, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

// Update replaces this connection in the terminal server list.
func (r *terminalServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing terminal servers.")
		return
	}

	var plan terminalServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn := expandTerminalServer(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	stored, err := r.client.UpdateTerminalServerConnection(ctx, conn)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Terminal server is gone",
				"Open WebUI no longer has a terminal server connection with the id "+conn.ID+". Something outside Terraform removed it. Run `terraform apply` again to recreate it.",
			)
			return
		}
		resp.Diagnostics.AddError("Update terminal server failed", err.Error())
		return
	}

	state := flattenTerminalServer(stored, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes this connection from the terminal server list.
func (r *terminalServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing terminal servers.")
		return
	}

	var state terminalServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteTerminalServerConnection(ctx, state.ServerID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Delete terminal server failed", err.Error())
	}
}

// ImportState takes the connection id and reads the rest from the list.
func (r *terminalServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_id"), req, resp)
}

func expandTerminalServer(plan terminalServerModel, diags *diag.Diagnostics) client.TerminalServerConnection {
	conn := client.TerminalServerConnection{
		ID:         plan.ServerID.ValueString(),
		Name:       stringPtr(plan.Name),
		URL:        plan.URL.ValueString(),
		Path:       stringPtr(plan.Path),
		Key:        stringPtr(plan.Key),
		AuthType:   stringPtr(plan.AuthType),
		Config:     decodeOptionalJSON(plan.ConfigJSON, path.Root("config_json"), diags),
		ServerType: stringPtr(plan.ServerType),
		PolicyID:   stringPtr(plan.PolicyID),
	}

	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		enabled := plan.Enabled.ValueBool()
		conn.Enabled = &enabled
	}

	return conn
}

func flattenTerminalServer(conn *client.TerminalServerConnection, diags *diag.Diagnostics) terminalServerModel {
	configJSON, err := encodeOptionalJSON(conn.Config)
	if err != nil {
		diags.AddError("Unexpected terminal server response", "Unable to encode the config of terminal server "+conn.ID+": "+err.Error())
	}

	state := terminalServerModel{
		ID:         types.StringValue(conn.ID),
		ServerID:   types.StringValue(conn.ID),
		Name:       stringValueOrNull(conn.Name),
		Enabled:    types.BoolNull(),
		URL:        types.StringValue(conn.URL),
		Path:       stringValueOrNull(conn.Path),
		Key:        stringValueOrNull(conn.Key),
		AuthType:   stringValueOrNull(conn.AuthType),
		ConfigJSON: configJSON,
		ServerType: stringValueOrNull(conn.ServerType),
		PolicyID:   stringValueOrNull(conn.PolicyID),
	}

	if conn.Enabled != nil {
		state.Enabled = types.BoolValue(*conn.Enabled)
	}

	return state
}
