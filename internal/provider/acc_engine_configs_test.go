package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The two validators that mirror the checks the images update route runs.
var (
	regexpImageSize  = regexp.MustCompile("must be WIDTHxHEIGHT, auto, or empty")
	regexpImageSteps = regexp.MustCompile("Invalid Attribute Value")
)

// The tests in this file mutate the target Open WebUI instance's engine
// settings. Run them against the throwaway container, never against a live one.

// TestAccRAGEmbeddingConfigResource keeps the built-in embedding engine and
// checks that a second plan is empty.
func TestAccRAGEmbeddingConfigResource(t *testing.T) {
	config := testAccProviderConfig() + `
resource "openwebui_rag_embedding_config" "test" {
  rag_embedding_engine     = ""
  rag_embedding_model      = "sentence-transformers/all-MiniLM-L6-v2"
  rag_embedding_batch_size = 1
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_rag_embedding_config.test", "rag_embedding_engine", ""),
					resource.TestCheckResourceAttr("openwebui_rag_embedding_config.test", "rag_embedding_batch_size", "1"),
					resource.TestCheckResourceAttrSet("openwebui_rag_embedding_config.test", "enable_async_embedding"),
					resource.TestCheckResourceAttrSet("openwebui_rag_embedding_config.test", "id"),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccRAGEmbeddingConfigResource_OpenAIBlock writes an engine block and
// checks the key reads back, which is how the resource converges its state.
func TestAccRAGEmbeddingConfigResource_OpenAIBlock(t *testing.T) {
	config := testAccProviderConfig() + `
resource "openwebui_rag_embedding_config" "test" {
  rag_embedding_engine = "openai"
  rag_embedding_model  = "text-embedding-3-small"

  openai_config = {
    url = "https://api.openai.com/v1"
    key = "sk-acceptance-test"
  }
}`

	// A config resource has no remote delete, so the instance keeps whatever
	// the last step wrote. Leaving it on the openai engine would make every
	// later test that attaches a document call api.openai.com and fail.
	restore := testAccProviderConfig() + `
resource "openwebui_rag_embedding_config" "test" {
  rag_embedding_engine = ""
  rag_embedding_model  = "sentence-transformers/all-MiniLM-L6-v2"
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_rag_embedding_config.test", "openai_config.url", "https://api.openai.com/v1"),
					resource.TestCheckResourceAttr("openwebui_rag_embedding_config.test", "openai_config.key", "sk-acceptance-test"),
					resource.TestCheckNoResourceAttr("openwebui_rag_embedding_config.test", "ollama_config.url"),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
			{
				Config: restore,
				Check: resource.TestCheckResourceAttr(
					"openwebui_rag_embedding_config.test", "rag_embedding_engine", ""),
			},
		},
	})
}

// TestAccRAGConfigResource applies a small retrieval config and checks that a
// second plan is empty.
func TestAccRAGConfigResource(t *testing.T) {
	config := testAccProviderConfig() + `
resource "openwebui_rag_config" "test" {
  chunk_size    = 900
  chunk_overlap = 90
  top_k         = 4
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_rag_config.test", "chunk_size", "900"),
					resource.TestCheckResourceAttr("openwebui_rag_config.test", "chunk_overlap", "90"),
					resource.TestCheckResourceAttr("openwebui_rag_config.test", "top_k", "4"),
					resource.TestCheckResourceAttrSet("openwebui_rag_config.test", "rag_template"),
					// The write route answers without this key, so a resource
					// that trusted the response would record it as null.
					resource.TestCheckResourceAttrSet("openwebui_rag_config.test", "rag_reranking_batch_size"),
					resource.TestCheckNoResourceAttr("openwebui_rag_config.test", "web.web_search_engine"),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccRAGConfigResource_UnnamedKeysSurvive writes a key, then applies a
// configuration that names a different one. The update route writes the whole
// form it receives, so a request built from the plan alone would null the first.
func TestAccRAGConfigResource_UnnamedKeysSurvive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_rag_config" "first" {
  rag_template = "Answer from the context."
}`,
				Check: resource.TestCheckResourceAttr("openwebui_rag_config.first", "rag_template", "Answer from the context."),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_rag_config" "second" {
  chunk_size = 1100
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_rag_config.second", "chunk_size", "1100"),
					resource.TestCheckResourceAttr("openwebui_rag_config.second", "rag_template", "Answer from the context."),
				),
			},
		},
	})
}

// TestAccRAGConfigResource_FileLimitClears sets the empty string that removes a
// file limit. Open WebUI stores null for it, so the sentinel has to survive in
// state or every plan asks to clear the limit again.
func TestAccRAGConfigResource_FileLimitClears(t *testing.T) {
	config := testAccProviderConfig() + `
resource "openwebui_rag_config" "test" {
  file_max_size  = ""
  file_max_count = "5"
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_rag_config.test", "file_max_size", ""),
					resource.TestCheckResourceAttr("openwebui_rag_config.test", "file_max_count", "5"),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccRAGConfigResource_WebBlock writes the web block, which the update
// route assigns whole.
func TestAccRAGConfigResource_WebBlock(t *testing.T) {
	config := testAccProviderConfig() + `
resource "openwebui_rag_config" "test" {
  web = {
    enable_web_search             = true
    web_search_engine             = "duckduckgo"
    web_search_result_count       = 5
    web_search_domain_filter_list = []
    tavily_api_key                = "tvly-acceptance-test"
  }
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_rag_config.test", "web.enable_web_search", "true"),
					resource.TestCheckResourceAttr("openwebui_rag_config.test", "web.web_search_engine", "duckduckgo"),
					resource.TestCheckResourceAttr("openwebui_rag_config.test", "web.web_search_result_count", "5"),
					resource.TestCheckResourceAttr("openwebui_rag_config.test", "web.web_search_domain_filter_list.#", "0"),
					resource.TestCheckResourceAttr("openwebui_rag_config.test", "web.tavily_api_key", "tvly-acceptance-test"),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccRAGConfigResource_JSONAttribute writes one of the free-form dict keys.
func TestAccRAGConfigResource_JSONAttribute(t *testing.T) {
	config := testAccProviderConfig() + `
resource "openwebui_rag_config" "test" {
  docling_params = jsonencode({ do_ocr = true })
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr("openwebui_rag_config.test", "docling_params", `{"do_ocr":true}`),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccImagesConfigResource applies an image generation config and checks
// that a second plan is empty. A successful apply does not prove the engine is
// reachable: the update route swallows the error when it cannot reach
// Automatic1111.
func TestAccImagesConfigResource(t *testing.T) {
	config := testAccProviderConfig() + `
resource "openwebui_images_config" "test" {
  enable_image_generation = false
  image_generation_engine = "openai"
  image_generation_model  = "dall-e-3"
  image_size              = "1024x1024"
  image_steps             = 50
  images_openai_api_key   = "sk-acceptance-test"
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_images_config.test", "image_generation_engine", "openai"),
					resource.TestCheckResourceAttr("openwebui_images_config.test", "image_size", "1024x1024"),
					resource.TestCheckResourceAttr("openwebui_images_config.test", "images_openai_api_key", "sk-acceptance-test"),
					resource.TestCheckResourceAttrSet("openwebui_images_config.test", "comfyui_workflow"),
					resource.TestCheckResourceAttrSet("openwebui_images_config.test", "id"),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccImagesConfigResource_RejectsBadSize checks that the schema catches the
// size the update route would answer 400 for.
func TestAccImagesConfigResource_RejectsBadSize(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_images_config" "test" {
  image_size = "1024"
}`,
				ExpectError: regexpImageSize,
			},
		},
	})
}

// TestAccImagesConfigResource_RejectsNegativeSteps checks the other validator
// the update route enforces.
func TestAccImagesConfigResource_RejectsNegativeSteps(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_images_config" "test" {
  image_steps = -1
}`,
				ExpectError: regexpImageSteps,
			},
		},
	})
}

// TestAccImagesConfigResource_TrailingSlashConverges writes a ComfyUI base URL
// with a trailing slash. Open WebUI strips it before storing, so a provider that
// took the stored form back into state would plan the same change forever.
func TestAccImagesConfigResource_TrailingSlashConverges(t *testing.T) {
	config := testAccProviderConfig() + `
resource "openwebui_images_config" "test" {
  comfyui_base_url = "http://comfyui.invalid:8188/"
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				// State keeps the configured form. Terraform rejects an applied
				// value that differs from the planned one, so the trimming can
				// only live on the request.
				Check: resource.TestCheckResourceAttr("openwebui_images_config.test", "comfyui_base_url", "http://comfyui.invalid:8188/"),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccAudioConfigResource applies both audio blocks and checks that a second
// plan is empty. It keeps the speech to text engine off the empty string, which
// would make Open WebUI download a faster-whisper model.
func TestAccAudioConfigResource(t *testing.T) {
	config := testAccProviderConfig() + `
resource "openwebui_audio_config" "test" {
  tts = {
    engine         = "openai"
    model          = "tts-1"
    voice          = "alloy"
    openai_api_key = "sk-acceptance-tts"
  }

  stt = {
    engine         = "openai"
    model          = "whisper-1"
    openai_api_key = "sk-acceptance-stt"
  }
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_audio_config.test", "tts.engine", "openai"),
					resource.TestCheckResourceAttr("openwebui_audio_config.test", "tts.voice", "alloy"),
					resource.TestCheckResourceAttr("openwebui_audio_config.test", "tts.openai_api_key", "sk-acceptance-tts"),
					resource.TestCheckResourceAttr("openwebui_audio_config.test", "stt.model", "whisper-1"),
					// Carries a Pydantic default a partial request would reset.
					resource.TestCheckResourceAttrSet("openwebui_audio_config.test", "stt.openai_api_request_format"),
					resource.TestCheckResourceAttrSet("openwebui_audio_config.test", "stt.supported_content_types.#"),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccAudioConfigResource_UnnamedKeysSurvive writes a key, then applies a
// configuration that names a different one in the same block.
func TestAccAudioConfigResource_UnnamedKeysSurvive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_audio_config" "first" {
  tts = {
    model = "tts-1-hd"
  }
  stt = {
    model = "whisper-1"
  }
}`,
				Check: resource.TestCheckResourceAttr("openwebui_audio_config.first", "tts.model", "tts-1-hd"),
			},
			{
				Config: testAccProviderConfig() + `
resource "openwebui_audio_config" "second" {
  tts = {
    voice = "nova"
  }
  stt = {
    azure_region = "eastus"
  }
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_audio_config.second", "tts.voice", "nova"),
					resource.TestCheckResourceAttr("openwebui_audio_config.second", "tts.model", "tts-1-hd"),
					resource.TestCheckResourceAttr("openwebui_audio_config.second", "stt.model", "whisper-1"),
				),
			},
		},
	})
}

// TestAccAudioConfigResource_LocalWhisper switches speech to text to the
// in-process engine. Open WebUI downloads the model when it is not cached, so
// this apply can take minutes and the test only runs when asked for by name.
func TestAccAudioConfigResource_LocalWhisper(t *testing.T) {
	testAccRequireEnv(t, "OPENWEBUI_ACC_WHISPER")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "openwebui_audio_config" "test" {
  tts = {
    engine = "openai"
  }
  stt = {
    engine        = ""
    whisper_model = "tiny"
  }
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("openwebui_audio_config.test", "stt.engine", ""),
					resource.TestCheckResourceAttr("openwebui_audio_config.test", "stt.whisper_model", "tiny"),
				),
			},
		},
	})
}
