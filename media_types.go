package yikdwebclient

// BodyType identifies how WebHelper should encode its request body.
type BodyType string

const (
	BodyTypeNone       BodyType = "none"
	BodyTypeFormData   BodyType = "formdata"
	BodyTypeURLEncoded BodyType = "urlencoded"
	BodyTypeRaw        BodyType = "raw"
)

const (
	MediaTypeApplicationFormURLEncoded = "application/x-www-form-urlencoded"
	MediaTypeApplicationJSON           = "application/json"
	MediaTypeApplicationJSONPatch      = "application/json-patch+json"
	MediaTypeApplicationJSONSequence   = "application/json-seq"
	MediaTypeApplicationManifest       = "application/manifest+json"
	MediaTypeApplicationOctetStream    = "application/octet-stream"
	MediaTypeApplicationPDF            = "application/pdf"
	MediaTypeApplicationProblemJSON    = "application/problem+json"
	MediaTypeApplicationProblemXML     = "application/problem+xml"
	MediaTypeApplicationRTF            = "application/rtf"
	MediaTypeApplicationSOAP           = "application/soap+xml"
	MediaTypeApplicationWASM           = "application/wasm"
	MediaTypeApplicationXML            = "application/xml"
	MediaTypeApplicationXMLDTD         = "application/xml-dtd"
	MediaTypeApplicationXMLPatch       = "application/xml-patch+xml"
	MediaTypeApplicationZIP            = "application/zip"

	MediaTypeFontCollection = "font/collection"
	MediaTypeFontOTF        = "font/otf"
	MediaTypeFontSFNT       = "font/sfnt"
	MediaTypeFontTTF        = "font/ttf"
	MediaTypeFontWOFF       = "font/woff"
	MediaTypeFontWOFF2      = "font/woff2"

	MediaTypeImageAVIF = "image/avif"
	MediaTypeImageBMP  = "image/bmp"
	MediaTypeImageGIF  = "image/gif"
	MediaTypeImageIcon = "image/x-icon"
	MediaTypeImageJPEG = "image/jpeg"
	MediaTypeImagePNG  = "image/png"
	MediaTypeImageSVG  = "image/svg+xml"
	MediaTypeImageTIFF = "image/tiff"
	MediaTypeImageWEBP = "image/webp"

	MediaTypeMultipartByteRanges = "multipart/byteranges"
	MediaTypeMultipartFormData   = "multipart/form-data"

	MediaTypeTextCSS        = "text/css"
	MediaTypeTextCSV        = "text/csv"
	MediaTypeTextHTML       = "text/html"
	MediaTypeTextJavaScript = "text/javascript"
	MediaTypeTextMarkdown   = "text/markdown"
	MediaTypeTextPlain      = "text/plain"
	MediaTypeTextRichText   = "text/richtext"
	MediaTypeTextRTF        = "text/rtf"
	MediaTypeTextXML        = "text/xml"
)
