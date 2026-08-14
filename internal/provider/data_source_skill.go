package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ datasource.DataSource = &skillDataSource{}
var _ datasource.DataSourceWithConfigure = &skillDataSource{}

// skillDataSource exposes skill details.
type skillDataSource struct {
	client *client.Client
}

// skillDataSourceModel maps data source inputs and outputs.
type skillDataSourceModel struct {
	SkillID     types.String `tfsdk:"skill_id"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Content     types.String `tfsdk:"content"`
	Tags        types.List   `tfsdk:"tags"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	ReadGroups  types.List   `tfsdk:"read_groups"`
	WriteGroups types.List   `tfsdk:"write_groups"`
	PublicRead  types.Bool   `tfsdk:"public_read"`
	PublicWrite types.Bool   `tfsdk:"public_write"`
	UserID      types.String `tfsdk:"user_id"`
	CreatedAt   types.Int64  `tfsdk:"created_at"`
	UpdatedAt   types.Int64  `tfsdk:"updated_at"`
	WriteAccess types.Bool   `tfsdk:"write_access"`
}

// NewSkillDataSource constructs a new skill data source.
func NewSkillDataSource() datasource.DataSource {
	return &skillDataSource{}
}

// Metadata sets the data source type name.
func (d *skillDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_skill"
}

// Schema defines the skill data source schema.
func (d *skillDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing skill by its `skill_id`.",
		Attributes: map[string]schema.Attribute{
			"skill_id": schema.StringAttribute{
				Required:            true,
				Description:         "Identifier of the skill to look up.",
				MarkdownDescription: "Identifier of the skill to look up.",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Identifier assigned by Open WebUI.",
				MarkdownDescription: "Identifier assigned by Open WebUI.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				Description:         "Display name of the skill.",
				MarkdownDescription: "Display name of the skill.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				Description:         "Short description of the skill.",
				MarkdownDescription: "Short description of the skill.",
			},
			"content": schema.StringAttribute{
				Computed:            true,
				Description:         "Markdown body of the skill.",
				MarkdownDescription: "Markdown body of the skill.",
			},
			"tags": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Description:         "Tags applied to the skill.",
				MarkdownDescription: "Tags applied to the skill.",
			},
			"is_active": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the skill is available to models.",
				MarkdownDescription: "Whether the skill is available to models.",
			},
			"read_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Description:         "Read-access group names currently applied to this skill.",
				MarkdownDescription: "Read-access group names currently applied to this skill.",
			},
			"write_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Description:         "Write-access group names currently applied to this skill.",
				MarkdownDescription: "Write-access group names currently applied to this skill.",
			},
			"public_read": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether every signed-in user can read the skill.",
				MarkdownDescription: "Whether every signed-in user can read the skill.",
			},
			"public_write": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether every signed-in user can write the skill.",
				MarkdownDescription: "Whether every signed-in user can write the skill.",
			},
			"user_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Owner user identifier.",
				MarkdownDescription: "Owner user identifier.",
			},
			"created_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of creation.",
				MarkdownDescription: "Unix timestamp of creation.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of last update.",
				MarkdownDescription: "Unix timestamp of last update.",
			},
			"write_access": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the authenticated user has write access.",
				MarkdownDescription: "Whether the authenticated user has write access.",
			},
		},
	}
}

// Configure assigns the API client.
func (d *skillDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read retrieves skill details.
func (d *skillDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the skill data source.")
		return
	}

	var config skillDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.SkillID.IsUnknown() || config.SkillID.IsNull() || config.SkillID.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("skill_id"),
			"Missing skill identifier",
			"The skill_id argument must be supplied to query an existing skill.",
		)
		return
	}

	access, err := d.client.GetSkill(ctx, config.SkillID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.Diagnostics.AddAttributeError(
				path.Root("skill_id"),
				"Skill not found",
				"No Open WebUI skill was found with the supplied skill_id.",
			)
			return
		}
		resp.Diagnostics.AddError("Read skill failed", err.Error())
		return
	}

	skill, diags := skillResponseToModel(ctx, d.client, access, types.StringNull())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := skillDataSourceModel{
		SkillID:     skill.SkillID,
		ID:          skill.ID,
		Name:        skill.Name,
		Description: skill.Description,
		Content:     skill.Content,
		Tags:        skill.Tags,
		IsActive:    skill.IsActive,
		ReadGroups:  skill.ReadGroups,
		WriteGroups: skill.WriteGroups,
		PublicRead:  skill.PublicRead,
		PublicWrite: skill.PublicWrite,
		UserID:      skill.UserID,
		CreatedAt:   skill.CreatedAt,
		UpdatedAt:   skill.UpdatedAt,
		WriteAccess: skill.WriteAccess,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
