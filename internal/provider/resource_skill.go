package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/docktape/terraform-provider-openwebui/internal/client"
)

var _ resource.Resource = &skillResource{}
var _ resource.ResourceWithConfigure = &skillResource{}
var _ resource.ResourceWithImportState = &skillResource{}

// skillIDPattern rejects the identifiers Open WebUI would rewrite on create. The
// create handler lowercases the ID and replaces spaces with hyphens, so an ID
// holding either would apply as a different ID than the config asked for.
var skillIDPattern = regexp.MustCompile(`^[^A-Z ]+$`)

// skillResource manages Open WebUI skills.
type skillResource struct {
	client *client.Client
}

// skillResourceModel captures Terraform state for skills.
type skillResourceModel struct {
	ID          types.String `tfsdk:"id"`
	SkillID     types.String `tfsdk:"skill_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Content     types.String `tfsdk:"content"`
	Tags        types.List   `tfsdk:"tags"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	ReadGroups  types.List   `tfsdk:"read_groups"`
	WriteGroups types.List   `tfsdk:"write_groups"`
	ReadUsers   types.List   `tfsdk:"read_users"`
	WriteUsers  types.List   `tfsdk:"write_users"`
	PublicRead  types.Bool   `tfsdk:"public_read"`
	PublicWrite types.Bool   `tfsdk:"public_write"`
	UserID      types.String `tfsdk:"user_id"`
	CreatedAt   types.Int64  `tfsdk:"created_at"`
	UpdatedAt   types.Int64  `tfsdk:"updated_at"`
	WriteAccess types.Bool   `tfsdk:"write_access"`
}

// NewSkillResource constructs a new skill resource.
func NewSkillResource() resource.Resource {
	return &skillResource{}
}

// Metadata sets the resource type name.
func (r *skillResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_skill"
}

// Schema defines the skill resource schema.
func (r *skillResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a skill in Open WebUI. A skill is a Markdown document that models load as extra instructions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Identifier assigned by Open WebUI on create.",
				MarkdownDescription: "Identifier assigned by Open WebUI on create.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"skill_id": schema.StringAttribute{
				Required:            true,
				Description:         "Unique identifier for the skill, e.g. `code-review`. Must be lowercase and free of spaces, because Open WebUI rewrites anything else. Forces replacement.",
				MarkdownDescription: "Unique identifier for the skill, e.g. `code-review`. Must be lowercase and free of spaces, because Open WebUI rewrites anything else. Forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(skillIDPattern, "must be lowercase and must not contain spaces"),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Display name of the skill. Unique across the Open WebUI instance.",
				MarkdownDescription: "Display name of the skill. Unique across the Open WebUI instance.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Short description shown in the Open WebUI interface.",
				MarkdownDescription: "Short description shown in the Open WebUI interface.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"content": schema.StringAttribute{
				Required:            true,
				Description:         "Markdown body of the skill.",
				MarkdownDescription: "Markdown body of the skill.",
			},
			"tags": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "List of tags for categorising the skill.",
				MarkdownDescription: "List of tags for categorising the skill.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"is_active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether the skill is available to models. Defaults to `true`.",
				MarkdownDescription: "Whether the skill is available to models. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"read_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "List of group names or IDs granted read access.",
				MarkdownDescription: "List of group names or IDs granted read access.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"write_groups": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "List of group names or IDs granted write access.",
				MarkdownDescription: "List of group names or IDs granted write access.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"read_users": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "List of user email addresses or IDs granted read access.",
				MarkdownDescription: "List of user email addresses or IDs granted read access.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"write_users": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "List of user email addresses or IDs granted write access. Name the same address in `read_users` as well.",
				MarkdownDescription: "List of user email addresses or IDs granted write access. Name the same address in `read_users` as well.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"public_read": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether every signed-in user can read the skill. This is the sharing the web UI calls public.",
				MarkdownDescription: "Whether every signed-in user can read the skill. This is the sharing the web UI calls public.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"public_write": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Whether every signed-in user can edit the skill.",
				MarkdownDescription: "Whether every signed-in user can edit the skill.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"user_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Owner user identifier. Set by Open WebUI.",
				MarkdownDescription: "Owner user identifier. Set by Open WebUI.",
			},
			"created_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of creation. Set by Open WebUI.",
				MarkdownDescription: "Unix timestamp of creation. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.Int64Attribute{
				Computed:            true,
				Description:         "Unix timestamp of last update. Set by Open WebUI.",
				MarkdownDescription: "Unix timestamp of last update. Set by Open WebUI.",
			},
			"write_access": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the authenticated user has write access to this skill. Read-only; set by Open WebUI.",
				MarkdownDescription: "Whether the authenticated user has write access to this skill. Read-only; set by Open WebUI.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// Configure assigns the API client.
func (r *skillResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if client, ok := req.ProviderData.(*client.Client); ok {
		r.client = client
	}
}

// Create provisions a skill.
func (r *skillResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing skills.")
		return
	}

	var plan skillResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form, diags := skillFormFromPlan(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateSkill(ctx, form)
	if err != nil {
		resp.Diagnostics.AddError("Create skill failed", err.Error())
		return
	}

	// The create response carries no content, so the skill is read back for it.
	access, err := r.client.GetSkill(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Read skill after create failed", err.Error())
		return
	}

	state, stateDiags := skillResponseToModel(ctx, r.client, access, plan.Content)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes skill state.
func (r *skillResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing skills.")
		return
	}

	var state skillResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	access, err := r.client.GetSkill(ctx, state.ID.ValueString())
	if err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read skill failed", err.Error())
		return
	}

	updated, diags := skillResponseToModel(ctx, r.client, access, state.Content)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

// Update mutates skill properties.
func (r *skillResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing skills.")
		return
	}

	var plan skillResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	form, diags := skillFormFromPlan(ctx, r.client, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateSkill(ctx, plan.ID.ValueString(), form)
	if err != nil {
		resp.Diagnostics.AddError("Update skill failed", err.Error())
		return
	}

	access, err := r.client.GetSkill(ctx, updated.ID)
	if err != nil {
		resp.Diagnostics.AddError("Read skill after update failed", err.Error())
		return
	}

	state, stateDiags := skillResponseToModel(ctx, r.client, access, plan.Content)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes a skill.
func (r *skillResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing skills.")
		return
	}

	var state skillResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteSkill(ctx, state.ID.ValueString()); err != nil {
		if err == client.ErrNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Delete skill failed", err.Error())
		return
	}
}

// ImportState maps an import identifier onto the id attribute.
func (r *skillResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("skill_id"), req.ID)...)
}

func skillFormFromPlan(ctx context.Context, apiClient *client.Client, plan skillResourceModel) (client.SkillForm, diag.Diagnostics) {
	var diags diag.Diagnostics

	var description *string
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		value := plan.Description.ValueString()
		description = &value
	}

	var isActive *bool
	if !plan.IsActive.IsNull() && !plan.IsActive.IsUnknown() {
		value := plan.IsActive.ValueBool()
		isActive = &value
	}

	tags := expandStringList(ctx, plan.Tags, path.Root("tags"), &diags)

	principals := resolveAccessPrincipals(ctx, apiClient, accessPrincipalLists{
		ReadGroups:  plan.ReadGroups,
		WriteGroups: plan.WriteGroups,
		ReadUsers:   plan.ReadUsers,
		WriteUsers:  plan.WriteUsers,
	}, &diags)

	// An unset public-sharing flag means the skill is not shared.
	publicRead := !plan.PublicRead.IsNull() && !plan.PublicRead.IsUnknown() && plan.PublicRead.ValueBool()
	publicWrite := !plan.PublicWrite.IsNull() && !plan.PublicWrite.IsUnknown() && plan.PublicWrite.ValueBool()

	accessControl := withPublicAccess(buildAccessControl(principals), publicRead, publicWrite)

	return client.SkillForm{
		ID:            plan.SkillID.ValueString(),
		Name:          plan.Name.ValueString(),
		Description:   description,
		Content:       plan.Content.ValueString(),
		Meta:          client.SkillMeta{Tags: tags},
		IsActive:      isActive,
		AccessControl: accessControl,
	}, diags
}

func skillResponseToModel(ctx context.Context, apiClient *client.Client, access *client.SkillAccessResponse, fallbackContent types.String) (skillResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	if access == nil {
		diags.AddError("Missing skill response", "Skill details were not returned by the Open WebUI API.")
		return skillResourceModel{}, diags
	}

	shared, sharedDiags := flattenAccessPrincipals(ctx, apiClient, access.AccessControl)
	diags.Append(sharedDiags...)

	tagList, tagDiags := flattenStringSlice(ctx, access.Meta.Tags)
	diags.Append(tagDiags...)

	description := types.StringNull()
	if access.Description != nil {
		description = types.StringValue(*access.Description)
	}

	contentValue := types.StringNull()
	if access.Content != "" {
		contentValue = types.StringValue(access.Content)
	} else if !fallbackContent.IsNull() && !fallbackContent.IsUnknown() {
		contentValue = types.StringValue(fallbackContent.ValueString())
	}

	state := skillResourceModel{
		ID:          types.StringValue(access.ID),
		SkillID:     types.StringValue(access.ID),
		Name:        types.StringValue(access.Name),
		Description: description,
		Content:     contentValue,
		Tags:        tagList,
		IsActive:    types.BoolValue(access.IsActive),
		ReadGroups:  shared.ReadGroups,
		WriteGroups: shared.WriteGroups,
		ReadUsers:   shared.ReadUsers,
		WriteUsers:  shared.WriteUsers,
		PublicRead:  types.BoolValue(publicAccessFromControl(access.AccessControl, "read")),
		PublicWrite: types.BoolValue(publicAccessFromControl(access.AccessControl, "write")),
		UserID:      types.StringValue(access.UserID),
		CreatedAt:   types.Int64Value(access.CreatedAt),
		UpdatedAt:   types.Int64Value(access.UpdatedAt),
		WriteAccess: types.BoolNull(),
	}

	if access.WriteAccess != nil {
		state.WriteAccess = types.BoolValue(*access.WriteAccess)
	}

	return state, diags
}
