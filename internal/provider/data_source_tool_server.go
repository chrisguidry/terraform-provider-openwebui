package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &toolServerDataSource{}
var _ datasource.DataSourceWithConfigure = &toolServerDataSource{}

// toolServerDataSource reads one registered tool server.
type toolServerDataSource struct {
	client *client.Client
}

type toolServerDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	ServerID               types.String `tfsdk:"server_id"`
	URL                    types.String `tfsdk:"url"`
	Path                   types.String `tfsdk:"path"`
	Type                   types.String `tfsdk:"type"`
	AuthType               types.String `tfsdk:"auth_type"`
	Key                    types.String `tfsdk:"key"`
	HeadersJSON            types.String `tfsdk:"headers_json"`
	SpecType               types.String `tfsdk:"spec_type"`
	Enabled                types.Bool   `tfsdk:"enabled"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	OAuthScope             types.String `tfsdk:"oauth_scope"`
	OAuthResourceParameter types.String `tfsdk:"oauth_resource_parameter"`
	OAuthClientID          types.String `tfsdk:"oauth_client_id"`
	ReadGroups             types.List   `tfsdk:"read_groups"`
	WriteGroups            types.List   `tfsdk:"write_groups"`
	PublicRead             types.Bool   `tfsdk:"public_read"`
	PublicWrite            types.Bool   `tfsdk:"public_write"`
}

// NewToolServerDataSource constructs a new tool server data source.
func NewToolServerDataSource() datasource.DataSource {
	return &toolServerDataSource{}
}

// Metadata sets the data source type name.
func (d *toolServerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tool_server"
}

// Schema defines the tool server data source schema.
func (d *toolServerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads one registered tool server by its `server_id`.\n\n" +
			"The encrypted OAuth registration and the static client secret are not exposed here. Read them " +
			"from the `openwebui_tool_server` resource that manages the connection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Mirrors `server_id`.",
				MarkdownDescription: "Mirrors `server_id`.",
			},
			"server_id": schema.StringAttribute{
				Required:            true,
				Description:         "Identifier of the tool server, stored as `info.id`.",
				MarkdownDescription: "Identifier of the tool server, stored as `info.id`.",
			},
			"url": schema.StringAttribute{
				Computed:            true,
				Description:         "Base URL of the tool server.",
				MarkdownDescription: "Base URL of the tool server.",
			},
			"path": schema.StringAttribute{
				Computed:            true,
				Description:         "Path to the OpenAPI spec.",
				MarkdownDescription: "Path to the OpenAPI spec.",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				Description:         "Tool server type, `openapi` or `mcp`.",
				MarkdownDescription: "Tool server type, `openapi` or `mcp`.",
			},
			"auth_type": schema.StringAttribute{
				Computed:            true,
				Description:         "How Open WebUI authenticates to the server.",
				MarkdownDescription: "How Open WebUI authenticates to the server.",
			},
			"key": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				Description:         "Bearer token sent to the tool server. Sensitive.",
				MarkdownDescription: "Bearer token sent to the tool server. Sensitive.",
			},
			"headers_json": schema.StringAttribute{
				Computed:            true,
				Description:         "JSON object of extra HTTP headers.",
				MarkdownDescription: "JSON object of extra HTTP headers.",
			},
			"spec_type": schema.StringAttribute{
				Computed:            true,
				Description:         "Where an OpenAPI server's spec comes from.",
				MarkdownDescription: "Where an OpenAPI server's spec comes from.",
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether Open WebUI loads the server.",
				MarkdownDescription: "Whether Open WebUI loads the server.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				Description:         "Display name of an MCP server.",
				MarkdownDescription: "Display name of an MCP server.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				Description:         "Description of an MCP server.",
				MarkdownDescription: "Description of an MCP server.",
			},
			"oauth_scope": schema.StringAttribute{
				Computed:            true,
				Description:         "Space-separated OAuth scopes requested for this server.",
				MarkdownDescription: "Space-separated OAuth scopes requested for this server.",
			},
			"oauth_resource_parameter": schema.StringAttribute{
				Computed:            true,
				Description:         "How Open WebUI sends the OAuth resource indicator.",
				MarkdownDescription: "How Open WebUI sends the OAuth resource indicator.",
			},
			"oauth_client_id": schema.StringAttribute{
				Computed:            true,
				Description:         "OAuth client identifier used by a static OAuth connection.",
				MarkdownDescription: "OAuth client identifier used by a static OAuth connection.",
			},
			"read_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Description:         "Group names granted read access to the server's tools.",
				MarkdownDescription: "Group names granted read access to the server's tools.",
			},
			"write_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Description:         "Group names granted write access to the server's tools.",
				MarkdownDescription: "Group names granted write access to the server's tools.",
			},
			"public_read": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether every signed-in user can use the server's tools.",
				MarkdownDescription: "Whether every signed-in user can use the server's tools.",
			},
			"public_write": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether every signed-in user can manage the server's tools.",
				MarkdownDescription: "Whether every signed-in user can manage the server's tools.",
			},
		},
	}
}

// Configure assigns the API client.
func (d *toolServerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if apiClient, ok := req.ProviderData.(*client.Client); ok {
		d.client = apiClient
	}
}

// Read looks the connection up by server_id.
func (d *toolServerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before reading tool servers.")
		return
	}

	var config toolServerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := config.ServerID.ValueString()

	stored, err := d.client.GetToolServerConnection(ctx, client.ToolServerLocator{ID: serverID})
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			resp.Diagnostics.AddError(
				"Tool server not found",
				fmt.Sprintf("Open WebUI holds no tool server connection with the id %q.", serverID),
			)
			return
		}

		resp.Diagnostics.AddError("Read tool server failed", err.Error())

		return
	}

	seed := toolServerResourceModel{ServerID: types.StringValue(serverID)}

	state, diags := toolServerStateFromEntry(ctx, d.client, stored, seed)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result := toolServerDataSourceModel{
		ID:                     state.ID,
		ServerID:               state.ServerID,
		URL:                    state.URL,
		Path:                   state.Path,
		Type:                   state.Type,
		AuthType:               state.AuthType,
		Key:                    state.Key,
		HeadersJSON:            state.HeadersJSON,
		SpecType:               state.SpecType,
		Enabled:                state.Enabled,
		Name:                   state.Name,
		Description:            state.Description,
		OAuthScope:             state.OAuthScope,
		OAuthResourceParameter: state.OAuthResourceParameter,
		OAuthClientID:          state.OAuthClientID,
		ReadGroups:             state.ReadGroups,
		WriteGroups:            state.WriteGroups,
		PublicRead:             state.PublicRead,
		PublicWrite:            state.PublicWrite,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &result)...)
}
