package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &knowledgeResource{}
var _ resource.ResourceWithConfigure = &knowledgeResource{}
var _ resource.ResourceWithImportState = &knowledgeResource{}

// knowledgeResource implements the Terraform resource for Open WebUI knowledge bases.
type knowledgeResource struct {
	client *client.Client
}

// knowledgeResourceModel maps the resource schema data.
type knowledgeResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ReadGroups  types.List   `tfsdk:"read_groups"`
	WriteGroups types.List   `tfsdk:"write_groups"`
	ReadUsers   types.List   `tfsdk:"read_users"`
	WriteUsers  types.List   `tfsdk:"write_users"`
	PublicRead  types.Bool   `tfsdk:"public_read"`
	PublicWrite types.Bool   `tfsdk:"public_write"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	UserID      types.String `tfsdk:"user_id"`
}

// NewKnowledgeResource returns a new instance.
func NewKnowledgeResource() resource.Resource {
	return &knowledgeResource{}
}

// Metadata implements resource.Resource.
func (r *knowledgeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_knowledge"
}

// Schema describes the resource schema.
func (r *knowledgeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a knowledge base entry in Open WebUI.\n\nWith no groups and neither `public_read` nor `public_write`, the entry is visible to its owner and to admins only. Set `public_read = true` to share it with every signed-in user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "UUID assigned by Open WebUI on create.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable name of the knowledge base.",
			},
			"description": schema.StringAttribute{
				Required:    true,
				Description: "Description shown in the Open WebUI interface.",
			},
			"read_groups": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Description:   "List of group names or IDs granted read access.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"write_groups": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Description:   "List of group names or IDs granted write access. Groups here automatically receive read access too.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"read_users": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Description:   "List of user email addresses or IDs granted read access.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"write_users": schema.ListAttribute{
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				Description:   "List of user email addresses or IDs granted write access. Name the same address in `read_users` as well.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"public_read": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "When `true`, every signed-in user can read the knowledge base. This is what the Open WebUI interface calls public sharing.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"public_write": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "When `true`, every signed-in user can edit the knowledge base.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				Description:   "Creation date in `YYYY-MM-DD` format. Set by Open WebUI.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last-updated date in `YYYY-MM-DD` format. Set by Open WebUI.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier of the Open WebUI user that owns the knowledge entry. Set by Open WebUI.",
			},
		},
	}
}

// Configure receives provider configuration data.
func (r *knowledgeResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create handles the creation of the resource.
func (r *knowledgeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing knowledge resources.")
		return
	}

	var plan knowledgeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form := client.KnowledgeForm{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	principals := resolveAccessPrincipals(ctx, r.client, accessPrincipalLists{
		ReadGroups:  plan.ReadGroups,
		WriteGroups: plan.WriteGroups,
		ReadUsers:   plan.ReadUsers,
		WriteUsers:  plan.WriteUsers,
	}, &resp.Diagnostics)

	form.AccessControl = withPublicAccess(buildAccessControl(principals), plan.PublicRead.ValueBool(), plan.PublicWrite.ValueBool())
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateKnowledge(ctx, form)
	if err != nil {
		resp.Diagnostics.AddError("Create knowledge entry failed", err.Error())
		return
	}

	// Fetch the latest representation to populate computed fields consistently.
	current, err := r.client.GetKnowledge(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Read knowledge entry failed", err.Error())
		return
	}

	state, diags := knowledgeResponseToModel(ctx, r.client, *current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the Terraform state with the latest API data.
func (r *knowledgeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing knowledge resources.")
		return
	}

	var state knowledgeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.client.GetKnowledge(ctx, state.ID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Read knowledge entry failed", err.Error())
		return
	}

	updated, diags := knowledgeResponseToModel(ctx, r.client, *current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

// Update applies plan changes.
func (r *knowledgeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing knowledge resources.")
		return
	}

	var plan knowledgeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form := client.KnowledgeForm{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	principals := resolveAccessPrincipals(ctx, r.client, accessPrincipalLists{
		ReadGroups:  plan.ReadGroups,
		WriteGroups: plan.WriteGroups,
		ReadUsers:   plan.ReadUsers,
		WriteUsers:  plan.WriteUsers,
	}, &resp.Diagnostics)

	form.AccessControl = withPublicAccess(buildAccessControl(principals), plan.PublicRead.ValueBool(), plan.PublicWrite.ValueBool())
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateKnowledge(ctx, plan.ID.ValueString(), form)
	if err != nil {
		resp.Diagnostics.AddError("Update knowledge entry failed", err.Error())
		return
	}

	current, err := r.client.GetKnowledge(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read knowledge entry failed", err.Error())
		return
	}

	state, diags := knowledgeResponseToModel(ctx, r.client, *current)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the knowledge resource.
func (r *knowledgeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing knowledge resources.")
		return
	}

	var state knowledgeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteKnowledge(ctx, state.ID.ValueString()); err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Delete knowledge entry failed", err.Error())
		return
	}
}

// ImportState maps imported IDs to the id attribute.
func (r *knowledgeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// knowledgeResponseToModel maps API structures to Terraform state.
func knowledgeResponseToModel(ctx context.Context, apiClient *client.Client, resp client.KnowledgeFilesResponse) (knowledgeResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	shared, sharedDiags := readAccessPrincipals(ctx, apiClient, resp.AccessControl)
	diags.Append(sharedDiags...)

	model := knowledgeResourceModel{
		ID:          types.StringValue(resp.ID),
		Name:        types.StringValue(resp.Name),
		Description: types.StringValue(resp.Description),
		ReadGroups:  nullableAccessList(ctx, shared.ReadGroups, &diags),
		WriteGroups: nullableAccessList(ctx, shared.WriteGroups, &diags),
		ReadUsers:   nullableAccessList(ctx, shared.ReadUsers, &diags),
		WriteUsers:  nullableAccessList(ctx, shared.WriteUsers, &diags),
		PublicRead:  types.BoolValue(publicAccessFromControl(resp.AccessControl, "read")),
		PublicWrite: types.BoolValue(publicAccessFromControl(resp.AccessControl, "write")),
		CreatedAt:   formatDateValue(resp.CreatedAt),
		UpdatedAt:   formatDateValue(resp.UpdatedAt),
		UserID:      types.StringValue(resp.UserID),
	}

	return model, diags
}
