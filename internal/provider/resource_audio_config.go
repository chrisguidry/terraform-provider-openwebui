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

func init() {
	registeredResources = append(registeredResources, NewAudioConfigResource)
}

var _ resource.Resource = &audioConfigResource{}
var _ resource.ResourceWithConfigure = &audioConfigResource{}
var _ resource.ResourceWithImportState = &audioConfigResource{}

// audioConfigResource manages the text to speech and speech to text engines.
type audioConfigResource struct {
	client *client.Client
}

// audioConfigModel carries both halves of AudioConfigUpdateForm, which requires
// them together.
type audioConfigModel struct {
	ID  types.String         `tfsdk:"id"`
	TTS *audioTTSConfigModel `tfsdk:"tts"`
	STT *audioSTTConfigModel `tfsdk:"stt"`
}

// audioTTSConfigModel mirrors TTSConfigForm in backend/open_webui/routers/audio.py.
type audioTTSConfigModel struct {
	OpenAIAPIBaseURL        types.String `tfsdk:"openai_api_base_url"`
	OpenAIAPIKey            types.String `tfsdk:"openai_api_key"`
	OpenAIParams            types.String `tfsdk:"openai_params"`
	APIKey                  types.String `tfsdk:"api_key"`
	Engine                  types.String `tfsdk:"engine"`
	Model                   types.String `tfsdk:"model"`
	Voice                   types.String `tfsdk:"voice"`
	SplitOn                 types.String `tfsdk:"split_on"`
	AzureSpeechRegion       types.String `tfsdk:"azure_speech_region"`
	AzureSpeechBaseURL      types.String `tfsdk:"azure_speech_base_url"`
	AzureSpeechOutputFormat types.String `tfsdk:"azure_speech_output_format"`
	MistralAPIKey           types.String `tfsdk:"mistral_api_key"`
	MistralAPIBaseURL       types.String `tfsdk:"mistral_api_base_url"`
}

// audioSTTConfigModel mirrors STTConfigForm in backend/open_webui/routers/audio.py.
type audioSTTConfigModel struct {
	OpenAIAPIBaseURL          types.String `tfsdk:"openai_api_base_url"`
	OpenAIAPIKey              types.String `tfsdk:"openai_api_key"`
	OpenAIAPIRequestFormat    types.String `tfsdk:"openai_api_request_format"`
	Engine                    types.String `tfsdk:"engine"`
	Model                     types.String `tfsdk:"model"`
	SupportedContentTypes     types.List   `tfsdk:"supported_content_types"`
	AllowedExtensions         types.List   `tfsdk:"allowed_extensions"`
	WhisperModel              types.String `tfsdk:"whisper_model"`
	DeepgramAPIKey            types.String `tfsdk:"deepgram_api_key"`
	AzureAPIKey               types.String `tfsdk:"azure_api_key"`
	AzureRegion               types.String `tfsdk:"azure_region"`
	AzureLocales              types.String `tfsdk:"azure_locales"`
	AzureBaseURL              types.String `tfsdk:"azure_base_url"`
	AzureMaxSpeakers          types.String `tfsdk:"azure_max_speakers"`
	MistralAPIKey             types.String `tfsdk:"mistral_api_key"`
	MistralAPIBaseURL         types.String `tfsdk:"mistral_api_base_url"`
	MistralUseChatCompletions types.Bool   `tfsdk:"mistral_use_chat_completions"`
}

// NewAudioConfigResource constructs a new audio config resource.
func NewAudioConfigResource() resource.Resource {
	return &audioConfigResource{}
}

// Metadata sets the resource type name.
func (r *audioConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_audio_config"
}

// Schema defines the audio config schema.
func (r *audioConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Open WebUI's audio engines: text to speech across OpenAI, Azure Speech, and Mistral, and speech to text " +
			"across local faster-whisper, OpenAI, Deepgram, Azure, and Mistral.\n\n" +
			"`POST /api/v1/audio/config/update` requires both blocks and writes every key of each, so this resource reads the current " +
			"configuration and overlays the attributes the plan names before it writes.\n\n" +
			"Setting `stt.engine` to the empty string makes Open WebUI load faster-whisper into its own process, downloading the model " +
			"named by `stt.whisper_model` when it is not already cached. That apply can take minutes on its first run.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Singleton identifier. Set by Open WebUI.",
				MarkdownDescription: "Singleton identifier. Set by Open WebUI.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tts": schema.SingleNestedAttribute{
				Required:            true,
				Description:         "Text to speech settings.",
				MarkdownDescription: "Text to speech settings.",
				Attributes: map[string]schema.Attribute{
					"openai_api_base_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `OPENAI_API_BASE_URL`. Stored as `audio.tts.openai.api_base_url`.",
						MarkdownDescription: "Open WebUI setting `OPENAI_API_BASE_URL`. Stored as `audio.tts.openai.api_base_url`.",
					},
					"openai_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `OPENAI_API_KEY`. Stored as `audio.tts.openai.api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `OPENAI_API_KEY`. Stored as `audio.tts.openai.api_key`. Sensitive.",
					},
					"openai_params": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "JSON object of extra parameters passed to the OpenAI speech API. Stored as `audio.tts.openai.params`.",
						MarkdownDescription: "JSON object of extra parameters passed to the OpenAI speech API. Stored as `audio.tts.openai.params`.",
					},
					"api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Key for the generic text to speech engine, used by ElevenLabs and others. Stored as `audio.tts.api_key`. Sensitive.",
						MarkdownDescription: "Key for the generic text to speech engine, used by ElevenLabs and others. Stored as `audio.tts.api_key`. Sensitive.",
					},
					"engine": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Text to speech engine. The empty string is Open WebUI's built-in engine. Stored as `audio.tts.engine`.",
						MarkdownDescription: "Text to speech engine. The empty string is Open WebUI's built-in engine. Stored as `audio.tts.engine`.",
					},
					"model": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `MODEL`. Stored as `audio.tts.model`.",
						MarkdownDescription: "Open WebUI setting `MODEL`. Stored as `audio.tts.model`.",
					},
					"voice": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `VOICE`. Stored as `audio.tts.voice`.",
						MarkdownDescription: "Open WebUI setting `VOICE`. Stored as `audio.tts.voice`.",
					},
					"split_on": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Boundary the text is split on before synthesis, for example `punctuation`. Stored as `audio.tts.split_on`.",
						MarkdownDescription: "Boundary the text is split on before synthesis, for example `punctuation`. Stored as `audio.tts.split_on`.",
					},
					"azure_speech_region": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `AZURE_SPEECH_REGION`. Stored as `audio.tts.azure.speech_region`.",
						MarkdownDescription: "Open WebUI setting `AZURE_SPEECH_REGION`. Stored as `audio.tts.azure.speech_region`.",
					},
					"azure_speech_base_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `AZURE_SPEECH_BASE_URL`. Stored as `audio.tts.azure.speech_base_url`.",
						MarkdownDescription: "Open WebUI setting `AZURE_SPEECH_BASE_URL`. Stored as `audio.tts.azure.speech_base_url`.",
					},
					"azure_speech_output_format": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `AZURE_SPEECH_OUTPUT_FORMAT`. Stored as `audio.tts.azure.speech_output_format`.",
						MarkdownDescription: "Open WebUI setting `AZURE_SPEECH_OUTPUT_FORMAT`. Stored as `audio.tts.azure.speech_output_format`.",
					},
					"mistral_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `MISTRAL_API_KEY`. Stored as `audio.tts.mistral.api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `MISTRAL_API_KEY`. Stored as `audio.tts.mistral.api_key`. Sensitive.",
					},
					"mistral_api_base_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `MISTRAL_API_BASE_URL`. Stored as `audio.tts.mistral.api_base_url`.",
						MarkdownDescription: "Open WebUI setting `MISTRAL_API_BASE_URL`. Stored as `audio.tts.mistral.api_base_url`.",
					},
				},
			},
			"stt": schema.SingleNestedAttribute{
				Required:            true,
				Description:         "Speech to text settings.",
				MarkdownDescription: "Speech to text settings.",
				Attributes: map[string]schema.Attribute{
					"openai_api_base_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `OPENAI_API_BASE_URL`. Stored as `audio.stt.openai.api_base_url`.",
						MarkdownDescription: "Open WebUI setting `OPENAI_API_BASE_URL`. Stored as `audio.stt.openai.api_base_url`.",
					},
					"openai_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `OPENAI_API_KEY`. Stored as `audio.stt.openai.api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `OPENAI_API_KEY`. Stored as `audio.stt.openai.api_key`. Sensitive.",
					},
					"openai_api_request_format": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Request encoding for the OpenAI transcription API, for example `multipart`. Stored as `audio.stt.openai.api_request_format`.",
						MarkdownDescription: "Request encoding for the OpenAI transcription API, for example `multipart`. Stored as `audio.stt.openai.api_request_format`.",
					},
					"engine": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Speech to text engine. The empty string runs faster-whisper in the Open WebUI process. Stored as `audio.stt.engine`.",
						MarkdownDescription: "Speech to text engine. The empty string runs faster-whisper in the Open WebUI process. Stored as `audio.stt.engine`.",
					},
					"model": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `MODEL`. Stored as `audio.stt.model`.",
						MarkdownDescription: "Open WebUI setting `MODEL`. Stored as `audio.stt.model`.",
					},
					"supported_content_types": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
						Description:         "Content types accepted for transcription. Stored as `audio.stt.supported_content_types`.",
						MarkdownDescription: "Content types accepted for transcription. Stored as `audio.stt.supported_content_types`.",
					},
					"allowed_extensions": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
						Description:         "File extensions accepted for transcription. Stored as `audio.stt.allowed_extensions`.",
						MarkdownDescription: "File extensions accepted for transcription. Stored as `audio.stt.allowed_extensions`.",
					},
					"whisper_model": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "faster-whisper model loaded when `engine` is the empty string. Stored as `audio.stt.whisper_model`.",
						MarkdownDescription: "faster-whisper model loaded when `engine` is the empty string. Stored as `audio.stt.whisper_model`.",
					},
					"deepgram_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `DEEPGRAM_API_KEY`. Stored as `audio.stt.deepgram.api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `DEEPGRAM_API_KEY`. Stored as `audio.stt.deepgram.api_key`. Sensitive.",
					},
					"azure_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `AZURE_API_KEY`. Stored as `audio.stt.azure.api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `AZURE_API_KEY`. Stored as `audio.stt.azure.api_key`. Sensitive.",
					},
					"azure_region": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `AZURE_REGION`. Stored as `audio.stt.azure.region`.",
						MarkdownDescription: "Open WebUI setting `AZURE_REGION`. Stored as `audio.stt.azure.region`.",
					},
					"azure_locales": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `AZURE_LOCALES`. Stored as `audio.stt.azure.locales`.",
						MarkdownDescription: "Open WebUI setting `AZURE_LOCALES`. Stored as `audio.stt.azure.locales`.",
					},
					"azure_base_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `AZURE_BASE_URL`. Stored as `audio.stt.azure.base_url`.",
						MarkdownDescription: "Open WebUI setting `AZURE_BASE_URL`. Stored as `audio.stt.azure.base_url`.",
					},
					"azure_max_speakers": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `AZURE_MAX_SPEAKERS`. Stored as `audio.stt.azure.max_speakers`.",
						MarkdownDescription: "Open WebUI setting `AZURE_MAX_SPEAKERS`. Stored as `audio.stt.azure.max_speakers`.",
					},
					"mistral_api_key": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
						Description:         "Open WebUI setting `MISTRAL_API_KEY`. Stored as `audio.stt.mistral.api_key`. Sensitive.",
						MarkdownDescription: "Open WebUI setting `MISTRAL_API_KEY`. Stored as `audio.stt.mistral.api_key`. Sensitive.",
					},
					"mistral_api_base_url": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `MISTRAL_API_BASE_URL`. Stored as `audio.stt.mistral.api_base_url`.",
						MarkdownDescription: "Open WebUI setting `MISTRAL_API_BASE_URL`. Stored as `audio.stt.mistral.api_base_url`.",
					},
					"mistral_use_chat_completions": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Open WebUI setting `MISTRAL_USE_CHAT_COMPLETIONS`. Stored as `audio.stt.mistral.use_chat_completions`.",
						MarkdownDescription: "Open WebUI setting `MISTRAL_USE_CHAT_COMPLETIONS`. Stored as `audio.stt.mistral.use_chat_completions`.",
					},
				},
			},
		},
	}
}

// Configure assigns the API client.
func (r *audioConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	if apiClient, ok := req.ProviderData.(*client.Client); ok {
		r.client = apiClient
	}
}

// Create writes the planned the audio config.
func (r *audioConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing the audio config.")
		return
	}

	var plan audioConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := applyAudioConfig(ctx, r.client, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes the the audio config from Open WebUI.
func (r *audioConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing the audio config.")
		return
	}

	var recorded audioConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &recorded)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := readAudioConfig(ctx, r.client, recorded, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update writes the planned the audio config.
func (r *audioConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured API client", "Expected provider to configure the Open WebUI client before managing the audio config.")
		return
	}

	var plan audioConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := applyAudioConfig(ctx, r.client, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete drops the resource from state and leaves Open WebUI's settings alone.
// There is no route that restores a default, and a config surface outlives the
// Terraform resource that describes it.
func (r *audioConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// ImportState maps import identifiers onto the id attribute.
func (r *audioConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyAudioConfig overlays the planned attributes on the current configuration
// and writes both blocks whole. Four keys carry Pydantic defaults that a
// partial request would silently reset: tts.OPENAI_PARAMS,
// stt.OPENAI_API_REQUEST_FORMAT, stt.SUPPORTED_CONTENT_TYPES, and
// stt.ALLOWED_EXTENSIONS.
func applyAudioConfig(ctx context.Context, apiClient *client.Client, plan audioConfigModel, diags *diag.Diagnostics) audioConfigModel {
	current, err := apiClient.GetAudioConfig(ctx)
	if err != nil {
		diags.AddError("Read audio config failed", err.Error())
		return audioConfigModel{}
	}

	form := expandAudioConfig(ctx, plan, current, diags)
	if diags.HasError() {
		return audioConfigModel{}
	}

	updated, err := apiClient.SetAudioConfig(ctx, form)
	if err != nil {
		diags.AddError("Update audio config failed", err.Error())
		return audioConfigModel{}
	}

	return flattenAudioConfig(ctx, plan, updated, diags)
}

// readAudioConfig refreshes the recorded state from Open WebUI.
func readAudioConfig(ctx context.Context, apiClient *client.Client, recorded audioConfigModel, diags *diag.Diagnostics) audioConfigModel {
	current, err := apiClient.GetAudioConfig(ctx)
	if err != nil {
		diags.AddError("Read audio config failed", err.Error())
		return audioConfigModel{}
	}

	return flattenAudioConfig(ctx, recorded, current, diags)
}

// expandAudioConfig builds the request form, taking each field from the plan
// when the plan carries a value and from the current configuration otherwise.
func expandAudioConfig(ctx context.Context, plan audioConfigModel, current *client.AudioConfigForm, diags *diag.Diagnostics) client.AudioConfigForm {
	ttsRoot := path.Root("tts")
	sttRoot := path.Root("stt")

	form := client.AudioConfigForm{
		TTS: client.AudioTTSConfigForm{},
		STT: client.AudioSTTConfigForm{},
	}

	if plan.TTS == nil {
		plan.TTS = &audioTTSConfigModel{}
	}
	if plan.STT == nil {
		plan.STT = &audioSTTConfigModel{}
	}

	form.TTS.OpenAIAPIBaseURL = engineString(plan.TTS.OpenAIAPIBaseURL, current.TTS.OpenAIAPIBaseURL)
	form.TTS.OpenAIAPIKey = engineString(plan.TTS.OpenAIAPIKey, current.TTS.OpenAIAPIKey)
	form.TTS.OpenAIParams = engineJSON(plan.TTS.OpenAIParams, current.TTS.OpenAIParams, ttsRoot.AtName("openai_params"), diags)
	form.TTS.APIKey = engineString(plan.TTS.APIKey, current.TTS.APIKey)
	form.TTS.Engine = engineString(plan.TTS.Engine, current.TTS.Engine)
	form.TTS.Model = engineString(plan.TTS.Model, current.TTS.Model)
	form.TTS.Voice = engineString(plan.TTS.Voice, current.TTS.Voice)
	form.TTS.SplitOn = engineString(plan.TTS.SplitOn, current.TTS.SplitOn)
	form.TTS.AzureSpeechRegion = engineString(plan.TTS.AzureSpeechRegion, current.TTS.AzureSpeechRegion)
	form.TTS.AzureSpeechBaseURL = engineString(plan.TTS.AzureSpeechBaseURL, current.TTS.AzureSpeechBaseURL)
	form.TTS.AzureSpeechOutputFormat = engineString(plan.TTS.AzureSpeechOutputFormat, current.TTS.AzureSpeechOutputFormat)
	form.TTS.MistralAPIKey = engineString(plan.TTS.MistralAPIKey, current.TTS.MistralAPIKey)
	form.TTS.MistralAPIBaseURL = engineString(plan.TTS.MistralAPIBaseURL, current.TTS.MistralAPIBaseURL)

	form.STT.OpenAIAPIBaseURL = engineString(plan.STT.OpenAIAPIBaseURL, current.STT.OpenAIAPIBaseURL)
	form.STT.OpenAIAPIKey = engineString(plan.STT.OpenAIAPIKey, current.STT.OpenAIAPIKey)
	form.STT.OpenAIAPIRequestFormat = engineString(plan.STT.OpenAIAPIRequestFormat, current.STT.OpenAIAPIRequestFormat)
	form.STT.Engine = engineString(plan.STT.Engine, current.STT.Engine)
	form.STT.Model = engineString(plan.STT.Model, current.STT.Model)
	form.STT.SupportedContentTypes = engineStringList(ctx, plan.STT.SupportedContentTypes, current.STT.SupportedContentTypes, sttRoot.AtName("supported_content_types"), diags)
	form.STT.AllowedExtensions = engineStringList(ctx, plan.STT.AllowedExtensions, current.STT.AllowedExtensions, sttRoot.AtName("allowed_extensions"), diags)
	form.STT.WhisperModel = engineString(plan.STT.WhisperModel, current.STT.WhisperModel)
	form.STT.DeepgramAPIKey = engineString(plan.STT.DeepgramAPIKey, current.STT.DeepgramAPIKey)
	form.STT.AzureAPIKey = engineString(plan.STT.AzureAPIKey, current.STT.AzureAPIKey)
	form.STT.AzureRegion = engineString(plan.STT.AzureRegion, current.STT.AzureRegion)
	form.STT.AzureLocales = engineString(plan.STT.AzureLocales, current.STT.AzureLocales)
	form.STT.AzureBaseURL = engineString(plan.STT.AzureBaseURL, current.STT.AzureBaseURL)
	form.STT.AzureMaxSpeakers = engineString(plan.STT.AzureMaxSpeakers, current.STT.AzureMaxSpeakers)
	form.STT.MistralAPIKey = engineString(plan.STT.MistralAPIKey, current.STT.MistralAPIKey)
	form.STT.MistralAPIBaseURL = engineString(plan.STT.MistralAPIBaseURL, current.STT.MistralAPIBaseURL)
	form.STT.MistralUseChatCompletions = engineBool(plan.STT.MistralUseChatCompletions, current.STT.MistralUseChatCompletions)

	return form
}

// flattenAudioConfig records the configuration Open WebUI returned.
func flattenAudioConfig(ctx context.Context, prior audioConfigModel, remote *client.AudioConfigForm, diags *diag.Diagnostics) audioConfigModel {
	// Only the TTS block carries a JSON attribute, whose recorded text has to
	// survive a reply that reorders its keys.
	priorTTS := prior.TTS
	if priorTTS == nil {
		priorTTS = &audioTTSConfigModel{}
	}

	tts := &audioTTSConfigModel{}
	stt := &audioSTTConfigModel{}

	tts.OpenAIAPIBaseURL = stringValueOrNull(remote.TTS.OpenAIAPIBaseURL)
	tts.OpenAIAPIKey = stringValueOrNull(remote.TTS.OpenAIAPIKey)
	tts.OpenAIParams = engineJSONValue(priorTTS.OpenAIParams, remote.TTS.OpenAIParams, "openai_params", diags)
	tts.APIKey = stringValueOrNull(remote.TTS.APIKey)
	tts.Engine = stringValueOrNull(remote.TTS.Engine)
	tts.Model = stringValueOrNull(remote.TTS.Model)
	tts.Voice = stringValueOrNull(remote.TTS.Voice)
	tts.SplitOn = stringValueOrNull(remote.TTS.SplitOn)
	tts.AzureSpeechRegion = stringValueOrNull(remote.TTS.AzureSpeechRegion)
	tts.AzureSpeechBaseURL = stringValueOrNull(remote.TTS.AzureSpeechBaseURL)
	tts.AzureSpeechOutputFormat = stringValueOrNull(remote.TTS.AzureSpeechOutputFormat)
	tts.MistralAPIKey = stringValueOrNull(remote.TTS.MistralAPIKey)
	tts.MistralAPIBaseURL = stringValueOrNull(remote.TTS.MistralAPIBaseURL)

	stt.OpenAIAPIBaseURL = stringValueOrNull(remote.STT.OpenAIAPIBaseURL)
	stt.OpenAIAPIKey = stringValueOrNull(remote.STT.OpenAIAPIKey)
	stt.OpenAIAPIRequestFormat = stringValueOrNull(remote.STT.OpenAIAPIRequestFormat)
	stt.Engine = stringValueOrNull(remote.STT.Engine)
	stt.Model = stringValueOrNull(remote.STT.Model)
	stt.SupportedContentTypes = engineStringListValue(ctx, remote.STT.SupportedContentTypes, diags)
	stt.AllowedExtensions = engineStringListValue(ctx, remote.STT.AllowedExtensions, diags)
	stt.WhisperModel = stringValueOrNull(remote.STT.WhisperModel)
	stt.DeepgramAPIKey = stringValueOrNull(remote.STT.DeepgramAPIKey)
	stt.AzureAPIKey = stringValueOrNull(remote.STT.AzureAPIKey)
	stt.AzureRegion = stringValueOrNull(remote.STT.AzureRegion)
	stt.AzureLocales = stringValueOrNull(remote.STT.AzureLocales)
	stt.AzureBaseURL = stringValueOrNull(remote.STT.AzureBaseURL)
	stt.AzureMaxSpeakers = stringValueOrNull(remote.STT.AzureMaxSpeakers)
	stt.MistralAPIKey = stringValueOrNull(remote.STT.MistralAPIKey)
	stt.MistralAPIBaseURL = stringValueOrNull(remote.STT.MistralAPIBaseURL)
	stt.MistralUseChatCompletions = engineBoolValue(remote.STT.MistralUseChatCompletions)

	return audioConfigModel{
		ID:  types.StringValue("audio"),
		TTS: tts,
		STT: stt,
	}
}
