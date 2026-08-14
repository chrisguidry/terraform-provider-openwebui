package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &toolServerResource{}
var _ resource.ResourceWithConfigure = &toolServerResource{}
var _ resource.ResourceWithImportState = &toolServerResource{}

func init() {
	registeredResources = append(registeredResources, NewToolServerResource)
	registeredDataSources = append(registeredDataSources, NewToolServerDataSource)
}

// toolServerResource manages one entry of TOOL_SERVER_CONNECTIONS.
type toolServerResource struct {
	client *client.Client
}

type toolServerResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	ServerID               types.String `tfsdk:"server_id"`
	URL                    types.String `tfsdk:"url"`
	Path                   types.String `tfsdk:"path"`
	Type                   types.String `tfsdk:"type"`
	AuthType               types.String `tfsdk:"auth_type"`
	Key                    types.String `tfsdk:"key"`
	HeadersJSON            types.String `tfsdk:"headers_json"`
	SpecType               types.String `tfsdk:"spec_type"`
	Spec                   types.String `tfsdk:"spec"`
	Enabled                types.Bool   `tfsdk:"enabled"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	OAuthScope             types.String `tfsdk:"oauth_scope"`
	OAuthResourceParameter types.String `tfsdk:"oauth_resource_parameter"`
	OAuthClientID          types.String `tfsdk:"oauth_client_id"`
	OAuthClientSecret      types.String `tfsdk:"oauth_client_secret"`
	OAuthClientInfo        types.String `tfsdk:"oauth_client_info"`
	ReadGroups             types.List   `tfsdk:"read_groups"`
	WriteGroups            types.List   `tfsdk:"write_groups"`
	PublicRead             types.Bool   `tfsdk:"public_read"`
	PublicWrite            types.Bool   `tfsdk:"public_write"`
}

// NewToolServerResource constructs a new tool server resource.
func NewToolServerResource() resource.Resource {
	return &toolServerResource{}
}

// Metadata sets the resource type name.
func (r *toolServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tool_server"
}

// Schema defines the tool server schema.
func (r *toolServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Registers one external tool server, OpenAPI or MCP, with Open WebUI.\n\n" +
			"Open WebUI keeps every tool server in a single configuration list, `TOOL_SERVER_CONNECTIONS`, " +
			"and the only write route replaces that list as a whole. This resource reads the list, edits its " +
			"own entry, and writes the list back, under a lock that serialises the resources of one Terraform " +
			"run. It leaves every field of every other entry, and every field of its own entry that it does " +
			"not model, exactly as it found them.\n\n" +
			"~> **Note:** `openwebui_tool_server` and `openwebui_tool_servers_config` manage the same list. " +
			"Use one or the other, never both. `openwebui_tool_server` is the recommended one.\n\n" +
			"~> **Note:** The lock covers one process. Two `terraform apply` runs against the same instance " +
			"at the same time can lose a tool server.\n\n" +
			"## OAuth\n\n" +
			"An MCP server with `auth_type = \"oauth_2.1\"` authenticates with a registration blob that Open " +
			"WebUI mints through dynamic client registration and stores encrypted in `info.oauth_client_info`. " +
			"Registering is a separate call, so the two resources compose:\n\n" +
			"```hcl\n" +
			"resource \"openwebui_oauth_client\" \"paperless\" {\n" +
			"  url       = \"https://paperless.mcp.example\"\n" +
			"  client_id = \"paperless\"\n" +
			"  type      = \"mcp\"\n" +
			"}\n\n" +
			"resource \"openwebui_tool_server\" \"paperless\" {\n" +
			"  server_id = \"paperless\"\n" +
			"  type      = \"mcp\"\n" +
			"  url       = \"https://paperless.mcp.example\"\n" +
			"  auth_type = \"oauth_2.1\"\n" +
			"  enabled   = true\n\n" +
			"  oauth_client_info = openwebui_oauth_client.paperless.oauth_client_info\n" +
			"}\n" +
			"```\n\n" +
			"The `client_id` of the OAuth client must equal the `server_id` of the tool server, because Open " +
			"WebUI looks the client up as `<type>:<server_id>`.\n\n" +
			"The blob is a Fernet ciphertext under `OAUTH_CLIENT_INFO_ENCRYPTION_KEY`, which defaults to " +
			"`WEBUI_SECRET_KEY`. Terraform carries it and cannot read or re-create it. Rotating either key " +
			"invalidates every stored blob, and recovering means registering each OAuth client again.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Mirrors `server_id`.",
				MarkdownDescription: "Mirrors `server_id`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"server_id": schema.StringAttribute{
				Required: true,
				Description: "Identifier of the tool server, stored as `info.id`. Models reference the server " +
					"as `server:mcp:<server_id>` or `server:<server_id>` in their `tool_ids`, so changing it " +
					"breaks every model that points at this server. Forces replacement.",
				MarkdownDescription: "Identifier of the tool server, stored as `info.id`. Models reference the " +
					"server as `server:mcp:<server_id>` or `server:<server_id>` in their `tool_ids`, so changing " +
					"it breaks every model that points at this server. Forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url": schema.StringAttribute{
				Required:            true,
				Description:         "Base URL of the tool server, e.g. `https://tools.example`.",
				MarkdownDescription: "Base URL of the tool server, e.g. `https://tools.example`.",
			},
			"path": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Path to the OpenAPI spec, e.g. `openapi.json`, appended to `url` unless it is " +
					"itself a full URL. MCP servers ignore it. Defaults to empty.",
				MarkdownDescription: "Path to the OpenAPI spec, e.g. `openapi.json`, appended to `url` unless " +
					"it is itself a full URL. MCP servers ignore it. Defaults to empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Tool server type, `openapi` or `mcp`. Defaults to `openapi`.",
				MarkdownDescription: "Tool server type, `openapi` or `mcp`. Defaults to `openapi`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"auth_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "How Open WebUI authenticates to the server: `none`, `bearer`, `oauth_2.1`, or " +
					"`oauth_2.1_static`. `bearer` reads `key`. Both OAuth types read the registration in " +
					"`oauth_client_info` and apply only to MCP servers.",
				MarkdownDescription: "How Open WebUI authenticates to the server: `none`, `bearer`, " +
					"`oauth_2.1`, or `oauth_2.1_static`. `bearer` reads `key`. Both OAuth types read the " +
					"registration in `oauth_client_info` and apply only to MCP servers.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				Description:         "Bearer token sent to the tool server when `auth_type` is `bearer`. Sensitive.",
				MarkdownDescription: "Bearer token sent to the tool server when `auth_type` is `bearer`. Sensitive.",
			},
			"headers_json": schema.StringAttribute{
				Optional:            true,
				Description:         "JSON object of extra HTTP headers, e.g. `jsonencode({ X-Api-Version = \"2\" })`.",
				MarkdownDescription: "JSON object of extra HTTP headers, e.g. `jsonencode({ X-Api-Version = \"2\" })`.",
			},
			"spec_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Where an OpenAPI server's spec comes from: `url` fetches it from `path`, `json` " +
					"reads it from `spec`.",
				MarkdownDescription: "Where an OpenAPI server's spec comes from: `url` fetches it from `path`, " +
					"`json` reads it from `spec`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"spec": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Inline OpenAPI spec as JSON text, read when `spec_type` is `json`.",
				MarkdownDescription: "Inline OpenAPI spec as JSON text, read when `spec_type` is `json`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "Whether Open WebUI loads the server. Stored as `config.enable`; a disabled " +
					"server stays registered and offers no tools. Defaults to `true`.",
				MarkdownDescription: "Whether Open WebUI loads the server. Stored as `config.enable`; a " +
					"disabled server stays registered and offers no tools. Defaults to `true`.",
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Display name of an MCP server, stored as `info.name`. An OpenAPI server takes " +
					"its name from the fetched spec instead.",
				MarkdownDescription: "Display name of an MCP server, stored as `info.name`. An OpenAPI server " +
					"takes its name from the fetched spec instead.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Description of an MCP server, stored as `info.description`.",
				MarkdownDescription: "Description of an MCP server, stored as `info.description`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"oauth_scope": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Space-separated OAuth scopes to request for this server.",
				MarkdownDescription: "Space-separated OAuth scopes to request for this server.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"oauth_resource_parameter": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "How Open WebUI sends the OAuth resource indicator: `auto`, `always`, or " +
					"`never`. An unrecognised value reads as `auto`.",
				MarkdownDescription: "How Open WebUI sends the OAuth resource indicator: `auto`, `always`, or " +
					"`never`. An unrecognised value reads as `auto`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"oauth_client_id": schema.StringAttribute{
				Optional: true,
				Description: "OAuth client identifier for `auth_type = \"oauth_2.1_static\"`, which overlays " +
					"it onto the registration blob.",
				MarkdownDescription: "OAuth client identifier for `auth_type = \"oauth_2.1_static\"`, which " +
					"overlays it onto the registration blob.",
			},
			"oauth_client_secret": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "OAuth client secret for `auth_type = \"oauth_2.1_static\"`. Stored in plain " +
					"text by Open WebUI. Sensitive.",
				MarkdownDescription: "OAuth client secret for `auth_type = \"oauth_2.1_static\"`. Stored in " +
					"plain text by Open WebUI. Sensitive.",
			},
			"oauth_client_info": schema.StringAttribute{
				Optional:  true,
				Computed:  true,
				Sensitive: true,
				Description: "Encrypted OAuth registration for the server, normally taken from the " +
					"`oauth_client_info` attribute of an `openwebui_oauth_client` resource. Left unset, the " +
					"blob already stored on the server is preserved untouched. Sensitive.",
				MarkdownDescription: "Encrypted OAuth registration for the server, normally taken from the " +
					"`oauth_client_info` attribute of an `openwebui_oauth_client` resource. Left unset, the " +
					"blob already stored on the server is preserved untouched. Sensitive.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"read_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "List of group names or IDs granted read access to the server's tools.",
				MarkdownDescription: "List of group names or IDs granted read access to the server's tools.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"write_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "List of group names or IDs granted write access to the server's tools.",
				MarkdownDescription: "List of group names or IDs granted write access to the server's tools.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"public_read": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether every signed-in user can use the server's tools.",
				MarkdownDescription: "Whether every signed-in user can use the server's tools.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"public_write": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether every signed-in user can manage the server's tools.",
				MarkdownDescription: "Whether every signed-in user can manage the server's tools.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure assigns the API client.
func (r *toolServerResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if apiClient, ok := req.ProviderData.(*client.Client); ok {
		r.client = apiClient
	}
}

// Create appends the connection to the tool server list.
func (r *toolServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tool servers.")
		return
	}

	var plan toolServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desired := toolServerEntryFromPlan(ctx, r.client, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	stored, err := r.client.CreateToolServerConnection(ctx, desired)
	if err != nil {
		if errors.Is(err, client.ErrToolServerExists) {
			resp.Diagnostics.AddAttributeError(
				path.Root("server_id"),
				"Tool server already registered",
				fmt.Sprintf("Open WebUI already holds a tool server connection with the id %q. Import it instead of creating it.", plan.ServerID.ValueString()),
			)
			return
		}

		resp.Diagnostics.AddError("Create tool server failed", err.Error())

		return
	}

	state, diags := toolServerStateFromEntry(ctx, r.client, stored, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes one connection from the tool server list.
func (r *toolServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tool servers.")
		return
	}

	var state toolServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stored, err := r.client.GetToolServerConnection(ctx, toolServerLocator(state))
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Read tool server failed", err.Error())

		return
	}

	refreshed, diags := toolServerStateFromEntry(ctx, r.client, stored, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

// Update edits the connection in place.
func (r *toolServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tool servers.")
		return
	}

	var plan toolServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state toolServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desired := toolServerEntryFromPlan(ctx, r.client, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	stored, err := r.client.UpsertToolServerConnection(ctx, toolServerLocator(state), desired)
	if err != nil {
		resp.Diagnostics.AddError("Update tool server failed", err.Error())
		return
	}

	updated, diags := toolServerStateFromEntry(ctx, r.client, stored, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

// Delete splices the connection out of the tool server list.
func (r *toolServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing tool servers.")
		return
	}

	var state toolServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteToolServerConnection(ctx, toolServerLocator(state)); err != nil {
		resp.Diagnostics.AddError("Delete tool server failed", err.Error())
	}
}

// ImportState adopts an existing connection.
//
// The import ID is the `server_id`, which Open WebUI stores as `info.id`. A
// connection made by hand in the web UI may carry no `info.id` at all, so the
// form `<server_id>@<index>` adopts the connection at a list position and names
// it. The next write pins `info.id` and the position stops mattering.
func (r *toolServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before importing tool servers.")
		return
	}

	serverID, index, err := parseToolServerImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	var stored client.ToolServerEntry
	if index >= 0 {
		stored, err = r.client.GetToolServerConnectionAt(ctx, index)
	} else {
		stored, err = r.client.GetToolServerConnection(ctx, client.ToolServerLocator{ID: serverID})
	}

	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Tool server not found",
				fmt.Sprintf("Open WebUI holds no tool server connection matching %q.", req.ID),
			)
			return
		}

		resp.Diagnostics.AddError("Import tool server failed", err.Error())

		return
	}

	seed := toolServerResourceModel{ServerID: types.StringValue(serverID)}

	state, diags := toolServerStateFromEntry(ctx, r.client, stored, seed)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// parseToolServerImportID splits an import ID into a server id and an optional
// list position. The position is -1 when the ID names no position.
func parseToolServerImportID(raw string) (string, int, error) {
	identifier := strings.TrimSpace(raw)
	if identifier == "" {
		return "", -1, fmt.Errorf("an import ID is required, either <server_id> or <server_id>@<index>")
	}

	serverID, position, found := strings.Cut(identifier, "@")
	if !found {
		return identifier, -1, nil
	}

	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		return "", -1, fmt.Errorf("the import ID %q names no server_id before the @", raw)
	}

	index, err := strconv.Atoi(strings.TrimSpace(position))
	if err != nil || index < 0 {
		return "", -1, fmt.Errorf("the import ID %q must end in a list position, e.g. weather@0", raw)
	}

	return serverID, index, nil
}

// toolServerLocator builds the locator that finds this resource's connection.
// The url and path carry the fallback for a connection adopted by position,
// which has no info.id until the next write.
func toolServerLocator(state toolServerResourceModel) client.ToolServerLocator {
	return client.ToolServerLocator{
		ID:   state.ServerID.ValueString(),
		URL:  state.URL.ValueString(),
		Path: state.Path.ValueString(),
	}
}

// toolServerEntryFromPlan builds the fields Terraform owns.
//
// A top-level field is always present, because Open WebUI declares auth_type,
// key and config as required fields even though they accept null. Inside info
// and config only the keys Terraform owns appear, and a key Terraform holds as
// null appears as nil, which the client removes. Every other key of the stored
// info and config survives untouched, which is what keeps the encrypted
// info.oauth_client_info alive through an edit.
func toolServerEntryFromPlan(ctx context.Context, apiClient *client.Client, plan toolServerResourceModel, diags *diag.Diagnostics) client.ToolServerEntry {
	serverType := "openapi"
	if value, ok := toolServerKnownString(plan.Type); ok {
		serverType = value
	}

	serverPath, _ := toolServerKnownString(plan.Path)

	entry := client.ToolServerEntry{
		"url":       plan.URL.ValueString(),
		"path":      serverPath,
		"type":      serverType,
		"auth_type": toolServerNullable(plan.AuthType),
		"key":       toolServerNullable(plan.Key),
		"headers":   decodeOptionalJSON(plan.HeadersJSON, path.Root("headers_json"), diags),
	}

	// spec_type and spec ride through as extra fields. A write that set them to
	// null would tell Open WebUI to read a spec from nowhere, so an unset
	// attribute leaves the stored value alone.
	if value, ok := toolServerKnownString(plan.SpecType); ok {
		entry["spec_type"] = value
	}
	if value, ok := toolServerKnownString(plan.Spec); ok {
		entry["spec"] = value
	}

	readNames := expandStringList(ctx, plan.ReadGroups, path.Root("read_groups"), diags)
	writeNames := expandStringList(ctx, plan.WriteGroups, path.Root("write_groups"), diags)
	readIDs := resolveGroupNamesToIDs(ctx, apiClient, readNames, path.Root("read_groups"), diags)
	writeIDs := resolveGroupNamesToIDs(ctx, apiClient, writeNames, path.Root("write_groups"), diags)

	publicRead := toolServerKnownBool(plan.PublicRead)
	publicWrite := toolServerKnownBool(plan.PublicWrite)
	accessControl := withPublicAccess(buildAccessControl(readIDs, writeIDs), publicRead, publicWrite)

	enabled := true
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		enabled = plan.Enabled.ValueBool()
	}

	entry["config"] = map[string]any{
		"enable":                   enabled,
		"access_grants":            client.ToolServerAccessGrants(accessControl),
		"oauth_scope":              toolServerNullable(plan.OAuthScope),
		"oauth_resource_parameter": toolServerNullable(plan.OAuthResourceParameter),
	}

	entry["info"] = map[string]any{
		"id":                       plan.ServerID.ValueString(),
		"name":                     toolServerNullable(plan.Name),
		"description":              toolServerNullable(plan.Description),
		"oauth_scope":              toolServerNullable(plan.OAuthScope),
		"oauth_resource_parameter": toolServerNullable(plan.OAuthResourceParameter),
		"oauth_client_id":          toolServerNullable(plan.OAuthClientID),
		"oauth_client_secret":      toolServerNullable(plan.OAuthClientSecret),
		"oauth_client_info":        toolServerNullable(plan.OAuthClientInfo),
	}

	return entry
}

// toolServerStateFromEntry maps a stored connection onto the resource model.
// The fallback supplies the server_id for a connection adopted by position,
// which carries no info.id yet.
func toolServerStateFromEntry(ctx context.Context, apiClient *client.Client, entry client.ToolServerEntry, fallback toolServerResourceModel) (toolServerResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	if entry == nil {
		diags.AddError("Missing tool server response", "The tool server connection was not returned by the Open WebUI API.")
		return toolServerResourceModel{}, diags
	}

	info := toolServerSubObject(entry, "info")
	config := toolServerSubObject(entry, "config")

	serverID := toolServerString(info, "id")
	if serverID == "" {
		serverID = fallback.ServerID.ValueString()
	}

	accessControl := client.ToolServerAccessControl(config["access_grants"])
	readIDs := extractGroupIDsFromAccessControl(accessControl, "read")
	writeIDs := extractGroupIDsFromAccessControl(accessControl, "write")

	readNames, readDiags := fetchGroupNamesForIDs(ctx, apiClient, readIDs)
	diags.Append(readDiags...)
	writeNames, writeDiags := fetchGroupNamesForIDs(ctx, apiClient, writeIDs)
	diags.Append(writeDiags...)

	readList, readListDiags := flattenStringSlice(ctx, readNames)
	diags.Append(readListDiags...)
	writeList, writeListDiags := flattenStringSlice(ctx, writeNames)
	diags.Append(writeListDiags...)

	headersJSON, err := encodeOptionalJSONValue(entry["headers"])
	if err != nil {
		diags.AddError("Encode tool server headers failed", err.Error())
	}
	headersJSON = toolServerPreservedJSON(fallback.HeadersJSON, headersJSON)

	serverType := toolServerString(entry, "type")
	if serverType == "" {
		serverType = "openapi"
	}

	enabled, _ := config["enable"].(bool)

	state := toolServerResourceModel{
		ID:                     types.StringValue(serverID),
		ServerID:               types.StringValue(serverID),
		URL:                    types.StringValue(toolServerString(entry, "url")),
		Path:                   types.StringValue(toolServerString(entry, "path")),
		Type:                   types.StringValue(serverType),
		AuthType:               toolServerStringValue(entry, "auth_type"),
		Key:                    toolServerStringValue(entry, "key"),
		HeadersJSON:            headersJSON,
		SpecType:               toolServerStringValue(entry, "spec_type"),
		Spec:                   toolServerStringValue(entry, "spec"),
		Enabled:                types.BoolValue(enabled),
		Name:                   toolServerStringValue(info, "name"),
		Description:            toolServerStringValue(info, "description"),
		OAuthScope:             toolServerPreferredString(info, config, "oauth_scope"),
		OAuthResourceParameter: toolServerPreferredString(info, config, "oauth_resource_parameter"),
		OAuthClientID:          toolServerStringValue(info, "oauth_client_id"),
		OAuthClientSecret:      toolServerStringValue(info, "oauth_client_secret"),
		OAuthClientInfo:        toolServerStringValue(info, "oauth_client_info"),
		ReadGroups:             readList,
		WriteGroups:            writeList,
		PublicRead:             types.BoolValue(publicAccessFromControl(accessControl, "read")),
		PublicWrite:            types.BoolValue(publicAccessFromControl(accessControl, "write")),
	}

	return state, diags
}

// toolServerPreservedJSON keeps the configured JSON text when it means the
// same thing as what the server returned. Terraform fails an apply whose
// result differs from the plan, and two JSON documents that differ only in the
// order of their keys are the same document.
func toolServerPreservedJSON(configured, stored types.String) types.String {
	if configured.IsNull() || configured.IsUnknown() || stored.IsNull() {
		return stored
	}

	var configuredValue, storedValue any
	if json.Unmarshal([]byte(configured.ValueString()), &configuredValue) != nil {
		return stored
	}
	if json.Unmarshal([]byte(stored.ValueString()), &storedValue) != nil {
		return stored
	}

	configuredCanonical, err := json.Marshal(configuredValue)
	if err != nil {
		return stored
	}
	storedCanonical, err := json.Marshal(storedValue)
	if err != nil {
		return stored
	}

	if string(configuredCanonical) != string(storedCanonical) {
		return stored
	}

	return configured
}

func toolServerSubObject(entry client.ToolServerEntry, key string) map[string]any {
	value, ok := entry[key].(map[string]any)
	if !ok {
		return map[string]any{}
	}

	return value
}

func toolServerString(values map[string]any, key string) string {
	value, _ := values[key].(string)

	return value
}

func toolServerStringValue(values map[string]any, key string) types.String {
	value, ok := values[key].(string)
	if !ok {
		return types.StringNull()
	}

	return types.StringValue(value)
}

// toolServerPreferredString reads a key Open WebUI accepts in two places. Its
// resolver reads info first and falls back to config, so the provider reports
// the value that is in force.
func toolServerPreferredString(info, config map[string]any, key string) types.String {
	if value, ok := info[key].(string); ok && value != "" {
		return types.StringValue(value)
	}

	if value, ok := config[key].(string); ok && value != "" {
		return types.StringValue(value)
	}

	return types.StringNull()
}

// toolServerNullable returns nil for a null or unknown attribute, which the client
// writes as a JSON null at the top level and as a removed key inside info and
// config.
func toolServerNullable(value types.String) any {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	return value.ValueString()
}

func toolServerKnownString(value types.String) (string, bool) {
	if value.IsNull() || value.IsUnknown() {
		return "", false
	}

	return value.ValueString(), true
}

func toolServerKnownBool(value types.Bool) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueBool()
}
