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

var _ datasource.DataSource = &knowledgeDataSource{}
var _ datasource.DataSourceWithConfigure = &knowledgeDataSource{}

// knowledgeDataSource surfaces knowledge base entries.
type knowledgeDataSource struct {
	client *client.Client
}

// knowledgeDataSourceModel embeds the resource fields and adds the lookup identifier.
type knowledgeDataSourceModel struct {
	KnowledgeID types.String `tfsdk:"knowledge_id"`
	FileCount   types.Int64  `tfsdk:"file_count"`
	knowledgeResourceModel
}

// NewKnowledgeDataSource creates a new knowledge data source instance.
func NewKnowledgeDataSource() datasource.DataSource {
	return &knowledgeDataSource{}
}

// Metadata sets the data source type name.
func (d *knowledgeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_knowledge"
}

// Schema describes the knowledge data source schema.
func (d *knowledgeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an existing knowledge base entry by name or ID.",
		Attributes: map[string]schema.Attribute{
			"knowledge_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "UUID of the knowledge base to look up directly. If omitted, `name` must be provided.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Name of the knowledge base to look up. Must match exactly (case-insensitive).",
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the knowledge base. Set after lookup.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Description returned by Open WebUI.",
			},
			"file_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of files attached to the knowledge base.",
			},
			"read_groups": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Read-access group names currently applied to this knowledge base.",
			},
			"write_groups": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Write-access group names currently applied to this knowledge base.",
			},
			"public_read": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether every signed-in user can read the knowledge base.",
			},
			"public_write": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether every signed-in user can edit the knowledge base.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Creation date in `YYYY-MM-DD` format.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last-updated date in `YYYY-MM-DD` format.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the Open WebUI account that owns this object.",
			},
		},
	}
}

// Configure attaches the provider client.
func (d *knowledgeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		d.client = client
	}
}

// Read looks up the requested knowledge entry.
func (d *knowledgeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before using the knowledge data source.")
		return
	}

	var config knowledgeDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := ""
	if !config.KnowledgeID.IsNull() && !config.KnowledgeID.IsUnknown() {
		id = strings.TrimSpace(config.KnowledgeID.ValueString())
	}

	name := ""
	if !config.Name.IsNull() && !config.Name.IsUnknown() {
		name = strings.TrimSpace(config.Name.ValueString())
	}

	if id == "" {
		if name == "" {
			resp.Diagnostics.AddError(
				"Missing knowledge lookup value",
				"Either knowledge_id or name must be provided to query an existing knowledge entry.",
			)
			return
		}

		entries, err := d.client.ListKnowledge(ctx)
		if err != nil {
			resp.Diagnostics.AddError("List knowledge entries failed", err.Error())
			return
		}

		var matches []client.KnowledgeListItem
		for _, entry := range entries {
			if strings.EqualFold(entry.Name, name) {
				matches = append(matches, entry)
			}
		}

		if len(matches) == 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("name"),
				"Knowledge entry not found",
				fmt.Sprintf("No Open WebUI knowledge entry was found with the name %q.", name),
			)
			return
		}
		if len(matches) > 1 {
			resp.Diagnostics.AddAttributeError(
				path.Root("name"),
				"Knowledge entry name not unique",
				fmt.Sprintf("Multiple knowledge entries share the name %q. Provide knowledge_id instead.", name),
			)
			return
		}

		id = matches[0].ID
	}

	current, err := d.client.GetKnowledge(ctx, id)
	if err != nil {
		if err == client.ErrNotFound {
			resp.Diagnostics.AddAttributeError(
				path.Root("knowledge_id"),
				"Knowledge entry not found",
				"No Open WebUI knowledge entry was found with the supplied knowledge_id.",
			)
			return
		}

		resp.Diagnostics.AddError("Read knowledge entry failed", err.Error())
		return
	}

	model, diags := knowledgeResponseToModel(ctx, d.client, *current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The knowledge detail endpoint answers with a null files key, so the count
	// comes from the file listing's total.
	fileCount, err := d.client.CountKnowledgeFiles(ctx, current.ID)
	if err != nil {
		resp.Diagnostics.AddError("Count knowledge files failed", err.Error())
		return
	}

	state := knowledgeDataSourceModel{
		KnowledgeID:            types.StringValue(current.ID),
		FileCount:              types.Int64Value(int64(fileCount)),
		knowledgeResourceModel: model,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
