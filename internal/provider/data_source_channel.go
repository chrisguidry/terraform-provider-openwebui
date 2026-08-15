package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &channelDataSource{}
var _ datasource.DataSourceWithConfigure = &channelDataSource{}

func init() {
	registeredDataSources = append(registeredDataSources, NewChannelDataSource)
}

// channelDataSource exposes channel details.
type channelDataSource struct {
	client *client.Client
}

type channelDataSourceModel struct {
	ChannelID   types.String `tfsdk:"channel_id"`
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
	Description types.String `tfsdk:"description"`
	IsPrivate   types.Bool   `tfsdk:"is_private"`
	DataJSON    types.String `tfsdk:"data_json"`
	MetaJSON    types.String `tfsdk:"meta_json"`
	ReadGroups  types.List   `tfsdk:"read_groups"`
	WriteGroups types.List   `tfsdk:"write_groups"`
	ReadUsers   types.List   `tfsdk:"read_users"`
	WriteUsers  types.List   `tfsdk:"write_users"`
	PublicRead  types.Bool   `tfsdk:"public_read"`
	PublicWrite types.Bool   `tfsdk:"public_write"`
	UserID      types.String `tfsdk:"user_id"`
	CreatedAt   types.Int64  `tfsdk:"created_at"`
	UpdatedAt   types.Int64  `tfsdk:"updated_at"`
}

// NewChannelDataSource constructs a new channel data source.
func NewChannelDataSource() datasource.DataSource {
	return &channelDataSource{}
}

// Metadata sets the data source type name.
func (d *channelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel"
}

// Schema defines the channel data source schema.
func (d *channelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an Open WebUI channel by ID or by name.",
		Attributes: map[string]schema.Attribute{
			"channel_id": schema.StringAttribute{
				Optional:    true,
				Description: "UUID of the channel to look up. Use this or `name`, not both.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Name of the channel to look up.",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the channel.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description shown under the channel name.",
			},
			"is_private": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the channel is private.",
			},
			"data_json": schema.StringAttribute{
				Computed:    true,
				Description: "Free-form channel data as a JSON object.",
			},
			"meta_json": schema.StringAttribute{
				Computed:    true,
				Description: "Free-form channel metadata as a JSON object.",
			},
			"read_groups": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Names of the groups whose members can read the channel.",
			},
			"write_groups": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Names of the groups whose members can post in the channel.",
			},
			"read_users": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Email addresses of the users who can read the channel.",
			},
			"write_users": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Email addresses of the users who can post in the channel.",
			},
			"public_read": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether every signed-in user can read the channel.",
			},
			"public_write": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether every signed-in user can post in the channel.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier of the account that created the channel.",
			},
			"created_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Creation timestamp in nanoseconds.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Last update timestamp in nanoseconds.",
			},
		},
	}
}

// Configure assigns the API client.
func (d *channelDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read retrieves channel details.
func (d *channelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the channel data source.")
		return
	}

	var config channelDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelID := ""
	if !config.ChannelID.IsNull() && !config.ChannelID.IsUnknown() {
		channelID = strings.TrimSpace(config.ChannelID.ValueString())
	}
	name := ""
	if !config.Name.IsNull() && !config.Name.IsUnknown() {
		name = strings.TrimSpace(config.Name.ValueString())
	}

	if channelID == "" && name == "" {
		resp.Diagnostics.AddError(
			"Missing channel lookup value",
			"Either channel_id or name must be provided to look up a channel.",
		)
		return
	}

	var channel *client.Channel
	if channelID != "" {
		found, err := d.client.GetChannel(ctx, channelID)
		if err != nil {
			if err == client.ErrNotFound {
				resp.Diagnostics.AddAttributeError(
					path.Root("channel_id"),
					"Channel not found",
					fmt.Sprintf("No Open WebUI channel was found with ID %q.", channelID),
				)
				return
			}
			resp.Diagnostics.AddError("Read channel failed", err.Error())
			return
		}
		channel = found
	} else {
		channels, err := d.client.ListChannels(ctx)
		if err != nil {
			resp.Diagnostics.AddError("List channels failed", err.Error())
			return
		}
		for i := range channels {
			if strings.EqualFold(channels[i].Name, name) {
				channel = &channels[i]
				break
			}
		}
		if channel == nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("name"),
				"Channel not found",
				fmt.Sprintf("No Open WebUI channel was found named %q.", name),
			)
			return
		}
	}

	shared, sharedDiags := flattenAccessPrincipals(ctx, d.client, channel.AccessControl)
	resp.Diagnostics.Append(sharedDiags...)

	dataJSON, dataErr := encodeOptionalJSON(channel.Data)
	if dataErr != nil {
		resp.Diagnostics.AddError("Serialize data_json failed", dataErr.Error())
	}
	metaJSON, metaErr := encodeOptionalJSON(channel.Meta)
	if metaErr != nil {
		resp.Diagnostics.AddError("Serialize meta_json failed", metaErr.Error())
	}

	if resp.Diagnostics.HasError() {
		return
	}

	description := types.StringNull()
	if channel.Description != nil {
		description = types.StringValue(*channel.Description)
	}

	isPrivate := types.BoolNull()
	if channel.IsPrivate != nil {
		isPrivate = types.BoolValue(*channel.IsPrivate)
	}

	state := channelDataSourceModel{
		ChannelID:   types.StringValue(channel.ID),
		Name:        types.StringValue(channel.Name),
		ID:          types.StringValue(channel.ID),
		Description: description,
		IsPrivate:   isPrivate,
		DataJSON:    dataJSON,
		MetaJSON:    metaJSON,
		ReadGroups:  shared.ReadGroups,
		WriteGroups: shared.WriteGroups,
		ReadUsers:   shared.ReadUsers,
		WriteUsers:  shared.WriteUsers,
		PublicRead:  types.BoolValue(publicAccessFromControl(channel.AccessControl, "read")),
		PublicWrite: types.BoolValue(publicAccessFromControl(channel.AccessControl, "write")),
		UserID:      types.StringValue(channel.UserID),
		CreatedAt:   types.Int64Value(channel.CreatedAt),
		UpdatedAt:   types.Int64Value(channel.UpdatedAt),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
