package snippets

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"strings"
)

//go:embed templates/*
var Templates embed.FS

type Framework string

const (
	FrameworkHTML    Framework = "html"
	FrameworkNextJS  Framework = "nextjs"
	FrameworkReact   Framework = "react"
	FrameworkVue     Framework = "vue"
	FrameworkSvelte  Framework = "svelte"
	FrameworkLaravel Framework = "laravel"
	FrameworkDjango  Framework = "django"
)

type Animation string

const (
	AnimationScramble   Animation = "scramble"
	AnimationPixel      Animation = "pixel"
	AnimationTypewriter Animation = "typewriter"
	AnimationNone       Animation = "none"
)

type Config struct {
	TestName      string
	Variants      []string
	ServerURL     string
	Animation     Animation
	WinnerVariant *int // nil if no winner, pointer to index if winner declared
}

type SnippetFile struct {
	Filename string
	Content  string
}

type templateData struct {
	TestName       string
	TestNamePascal string
	Variants       []string
	VariantsJSON   string
	VariantCount   int
	ServerURL      string
	Animation      string
	WinnerVariant  *int
	WinnerText     string
}

func Generate(framework Framework, config Config) ([]SnippetFile, error) {
	data := buildTemplateData(config)

	// If there's a winner, generate static content
	if config.WinnerVariant != nil {
		return generateStaticWinner(framework, data)
	}

	switch framework {
	case FrameworkHTML:
		return generateHTML(data)
	case FrameworkReact:
		return generateReact(data)
	case FrameworkNextJS:
		return generateNextJS(data)
	case FrameworkVue:
		return generateVue(data)
	case FrameworkSvelte:
		return generateSvelte(data)
	case FrameworkLaravel:
		return generateLaravel(data)
	case FrameworkDjango:
		return generateDjango(data)
	default:
		return generateHTML(data)
	}
}

func buildTemplateData(config Config) templateData {
	variantsJSON, _ := json.Marshal(config.Variants)

	winnerText := ""
	if config.WinnerVariant != nil && *config.WinnerVariant < len(config.Variants) {
		winnerText = config.Variants[*config.WinnerVariant]
	}

	return templateData{
		TestName:       config.TestName,
		TestNamePascal: toPascalCase(config.TestName),
		Variants:       config.Variants,
		VariantsJSON:   string(variantsJSON),
		VariantCount:   len(config.Variants),
		ServerURL:      config.ServerURL,
		Animation:      string(config.Animation),
		WinnerVariant:  config.WinnerVariant,
		WinnerText:     winnerText,
	}
}

func toPascalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	// Simple pascal case: capitalize first letter
	return strings.ToUpper(s[:1]) + s[1:]
}

func renderTemplate(name, content string, data templateData) (string, error) {
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func generateStaticWinner(framework Framework, data templateData) ([]SnippetFile, error) {
	content := `<!-- Static winner: "` + data.WinnerText + `" -->
<span>` + data.WinnerText + `</span>`

	return []SnippetFile{
		{Filename: "static-winner.html", Content: content},
	}, nil
}

func generateHTML(data templateData) ([]SnippetFile, error) {
	content := `<!-- headline-goat A/B Test: {{.TestName}} -->
<script src="{{.ServerURL}}/t/{{.TestName}}.js" defer></script>

<!-- Add this attribute to your headline element -->
<h1 data-ht-test="{{.TestName}}">{{index .Variants 0}}</h1>

<!-- Add this to your conversion button/link -->
<button data-ht-convert="{{.TestName}}">Get Started</button>
`

	rendered, err := renderTemplate("html", content, data)
	if err != nil {
		return nil, err
	}

	return []SnippetFile{
		{Filename: "headline-test.html", Content: rendered},
	}, nil
}

func generateReact(data templateData) ([]SnippetFile, error) {
	files := []SnippetFile{}

	// useVisitorId.ts
	useVisitorId := `import { useState, useEffect } from 'react';

export function useVisitorId(): string {
  const [visitorId, setVisitorId] = useState<string>('');

  useEffect(() => {
    let id = localStorage.getItem('ht_visitor_id');
    if (!id) {
      id = crypto.randomUUID();
      localStorage.setItem('ht_visitor_id', id);
    }
    setVisitorId(id);
  }, []);

  return visitorId;
}
`
	files = append(files, SnippetFile{Filename: "useVisitorId.ts", Content: useVisitorId})

	// useConvert.ts
	useConvert := `import { useCallback } from 'react';
import { useVisitorId } from './useVisitorId';

const SERVER_URL = '{{.ServerURL}}';

export function useConvert(testName: string) {
  const visitorId = useVisitorId();

  const convert = useCallback(() => {
    if (!visitorId) return;

    fetch(SERVER_URL + '/b', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        t: testName,
        v: parseInt(sessionStorage.getItem('ht_variant_' + testName) || '0'),
        e: 'convert',
        vid: visitorId,
      }),
    }).catch(() => {});
  }, [testName, visitorId]);

  return convert;
}
`
	rendered, _ := renderTemplate("useConvert", useConvert, data)
	files = append(files, SnippetFile{Filename: "useConvert.ts", Content: rendered})

	// TrackImpression.tsx
	trackImpression := `'use client';

import { useEffect, useState } from 'react';
import { useVisitorId } from './useVisitorId';

const SERVER_URL = '{{.ServerURL}}';
const TEST_NAME = '{{.TestName}}';
const VARIANTS: string[] = {{.VariantsJSON}};

interface Props {
  children?: React.ReactNode;
  className?: string;
}

export function TrackImpression({ children, className }: Props) {
  const visitorId = useVisitorId();
  const [variant, setVariant] = useState<number>(0);
  const [text, setText] = useState<string>('');
  const [ready, setReady] = useState(false);

  useEffect(() => {
    // Pick variant deterministically based on visitor ID
    const stored = sessionStorage.getItem('ht_variant_' + TEST_NAME);
    let v: number;
    if (stored !== null) {
      v = parseInt(stored);
    } else {
      v = Math.floor(Math.random() * VARIANTS.length);
      sessionStorage.setItem('ht_variant_' + TEST_NAME, v.toString());
    }
    setVariant(v);
    setText(VARIANTS[v]);
    setReady(true);

    // Track view
    if (visitorId) {
      fetch(SERVER_URL + '/b', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          t: TEST_NAME,
          v: v,
          e: 'view',
          vid: visitorId,
        }),
      }).catch(() => {});
    }
  }, [visitorId]);

  if (!ready) return null;

  return (
    <span className={className} data-ht-variant={variant}>
      {text}
      {children}
    </span>
  );
}
`
	rendered, _ = renderTemplate("TrackImpression", trackImpression, data)
	files = append(files, SnippetFile{Filename: "TrackImpression.tsx", Content: rendered})

	// ConvertButton.tsx
	convertButton := `'use client';

import { useConvert } from './useConvert';

interface Props {
  testName: string;
  children: React.ReactNode;
  className?: string;
  onClick?: () => void;
}

export function ConvertButton({ testName, children, className, onClick }: Props) {
  const convert = useConvert(testName);

  const handleClick = () => {
    convert();
    onClick?.();
  };

  return (
    <button className={className} onClick={handleClick}>
      {children}
    </button>
  );
}
`
	files = append(files, SnippetFile{Filename: "ConvertButton.tsx", Content: convertButton})

	// usage.tsx
	usage := `// Example usage in your component:
import { TrackImpression } from './TrackImpression';
import { ConvertButton } from './ConvertButton';

export default function HeroSection() {
  return (
    <div>
      <h1>
        <TrackImpression />
      </h1>
      <ConvertButton testName="{{.TestName}}">
        Get Started
      </ConvertButton>
    </div>
  );
}
`
	rendered, _ = renderTemplate("usage", usage, data)
	files = append(files, SnippetFile{Filename: "usage.tsx", Content: rendered})

	return files, nil
}

func generateNextJS(data templateData) ([]SnippetFile, error) {
	// Start with React files
	files, err := generateReact(data)
	if err != nil {
		return nil, err
	}

	// Add middleware.ts
	middleware := `import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

export function middleware(request: NextRequest) {
  const response = NextResponse.next();

  // Set visitor ID cookie if not present
  if (!request.cookies.get('ht_visitor_id')) {
    const visitorId = crypto.randomUUID();
    response.cookies.set('ht_visitor_id', visitorId, {
      httpOnly: false,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      maxAge: 60 * 60 * 24 * 365, // 1 year
    });
  }

  return response;
}

export const config = {
  matcher: '/((?!api|_next/static|_next/image|favicon.ico).*)',
};
`
	files = append([]SnippetFile{{Filename: "middleware.ts", Content: middleware}}, files...)

	return files, nil
}

func generateVue(data templateData) ([]SnippetFile, error) {
	content := `<template>
  <span :data-ht-variant="variant">{{"{{" }} text {{ "}}" }}</span>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';

const SERVER_URL = '{{.ServerURL}}';
const TEST_NAME = '{{.TestName}}';
const VARIANTS: string[] = {{.VariantsJSON}};

const variant = ref<number>(0);
const text = ref<string>('');

function getVisitorId(): string {
  let id = localStorage.getItem('ht_visitor_id');
  if (!id) {
    id = crypto.randomUUID();
    localStorage.setItem('ht_visitor_id', id);
  }
  return id;
}

onMounted(() => {
  const stored = sessionStorage.getItem('ht_variant_' + TEST_NAME);
  let v: number;
  if (stored !== null) {
    v = parseInt(stored);
  } else {
    v = Math.floor(Math.random() * VARIANTS.length);
    sessionStorage.setItem('ht_variant_' + TEST_NAME, v.toString());
  }

  variant.value = v;
  text.value = VARIANTS[v];

  // Track view
  const visitorId = getVisitorId();
  fetch(SERVER_URL + '/b', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      t: TEST_NAME,
      v: v,
      e: 'view',
      vid: visitorId,
    }),
  }).catch(() => {});
});

function convert() {
  const visitorId = getVisitorId();
  const v = parseInt(sessionStorage.getItem('ht_variant_' + TEST_NAME) || '0');

  fetch(SERVER_URL + '/b', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      t: TEST_NAME,
      v: v,
      e: 'convert',
      vid: visitorId,
    }),
  }).catch(() => {});
}

defineExpose({ convert });
</script>
`

	rendered, err := renderTemplate("vue", content, data)
	if err != nil {
		return nil, err
	}

	return []SnippetFile{
		{Filename: "HeadlineTest.vue", Content: rendered},
	}, nil
}

func generateSvelte(data templateData) ([]SnippetFile, error) {
	content := `<script lang="ts">
  import { onMount } from 'svelte';

  const SERVER_URL = '{{.ServerURL}}';
  const TEST_NAME = '{{.TestName}}';
  const VARIANTS: string[] = {{.VariantsJSON}};

  let variant = 0;
  let text = '';

  function getVisitorId(): string {
    let id = localStorage.getItem('ht_visitor_id');
    if (!id) {
      id = crypto.randomUUID();
      localStorage.setItem('ht_visitor_id', id);
    }
    return id;
  }

  onMount(() => {
    const stored = sessionStorage.getItem('ht_variant_' + TEST_NAME);
    let v: number;
    if (stored !== null) {
      v = parseInt(stored);
    } else {
      v = Math.floor(Math.random() * VARIANTS.length);
      sessionStorage.setItem('ht_variant_' + TEST_NAME, v.toString());
    }

    variant = v;
    text = VARIANTS[v];

    // Track view
    const visitorId = getVisitorId();
    fetch(SERVER_URL + '/b', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        t: TEST_NAME,
        v: v,
        e: 'view',
        vid: visitorId,
      }),
    }).catch(() => {});
  });

  export function convert() {
    const visitorId = getVisitorId();
    const v = parseInt(sessionStorage.getItem('ht_variant_' + TEST_NAME) || '0');

    fetch(SERVER_URL + '/b', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        t: TEST_NAME,
        v: v,
        e: 'convert',
        vid: visitorId,
      }),
    }).catch(() => {});
  }
</script>

<span data-ht-variant={variant}>{text}</span>
`

	rendered, err := renderTemplate("svelte", content, data)
	if err != nil {
		return nil, err
	}

	return []SnippetFile{
		{Filename: "HeadlineTest.svelte", Content: rendered},
	}, nil
}

func generateLaravel(data templateData) ([]SnippetFile, error) {
	files := []SnippetFile{}

	// Middleware
	middleware := `<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;
use Illuminate\Support\Str;

class HeadlineTestMiddleware
{
    public function handle(Request $request, Closure $next)
    {
        // Set visitor ID cookie if not present
        if (!$request->cookie('ht_visitor_id')) {
            $visitorId = Str::uuid()->toString();
            cookie()->queue('ht_visitor_id', $visitorId, 60 * 24 * 365);
        }

        return $next($request);
    }
}
`
	files = append(files, SnippetFile{Filename: "HeadlineTestMiddleware.php", Content: middleware})

	// Blade component for tracking
	trackBlade := `{{-- resources/views/components/track-impression.blade.php --}}
@props(['test' => '{{.TestName}}', 'variants' => {{.VariantsJSON}}])

@php
$variantIndex = session('ht_variant_' . $test);
if ($variantIndex === null) {
    $variantIndex = rand(0, count($variants) - 1);
    session(['ht_variant_' . $test => $variantIndex]);
}
$text = $variants[$variantIndex];
$visitorId = request()->cookie('ht_visitor_id') ?? '';
@endphp

<span data-ht-variant="{{ $variantIndex }}">{{ $text }}</span>

<script>
(function() {
    fetch('{{.ServerURL}}/b', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            t: '{{ $test }}',
            v: {{ $variantIndex }},
            e: 'view',
            vid: '{{ $visitorId }}'
        })
    }).catch(() => {});
})();
</script>
`
	rendered, _ := renderTemplate("trackBlade", trackBlade, data)
	files = append(files, SnippetFile{Filename: "track-impression.blade.php", Content: rendered})

	// Convert button component
	convertBlade := `{{-- resources/views/components/convert-button.blade.php --}}
@props(['test' => '{{.TestName}}'])

@php
$variantIndex = session('ht_variant_' . $test) ?? 0;
$visitorId = request()->cookie('ht_visitor_id') ?? '';
@endphp

<button {{ $attributes }} onclick="htConvert_{{ $test }}()">
    {{ $slot }}
</button>

<script>
function htConvert_{{ $test }}() {
    fetch('{{.ServerURL}}/b', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            t: '{{ $test }}',
            v: {{ $variantIndex }},
            e: 'convert',
            vid: '{{ $visitorId }}'
        })
    }).catch(() => {});
}
</script>
`
	rendered, _ = renderTemplate("convertBlade", convertBlade, data)
	files = append(files, SnippetFile{Filename: "convert-button.blade.php", Content: rendered})

	return files, nil
}

func generateDjango(data templateData) ([]SnippetFile, error) {
	files := []SnippetFile{}

	// Middleware
	middleware := `# middleware.py
import uuid
from django.utils.deprecation import MiddlewareMixin

class HeadlineTestMiddleware(MiddlewareMixin):
    def process_request(self, request):
        if 'ht_visitor_id' not in request.COOKIES:
            request.ht_visitor_id = str(uuid.uuid4())
        else:
            request.ht_visitor_id = request.COOKIES['ht_visitor_id']

    def process_response(self, request, response):
        if hasattr(request, 'ht_visitor_id') and 'ht_visitor_id' not in request.COOKIES:
            response.set_cookie('ht_visitor_id', request.ht_visitor_id, max_age=365*24*60*60)
        return response
`
	files = append(files, SnippetFile{Filename: "middleware.py", Content: middleware})

	// Template tags
	templateTags := `# templatetags/headline_test.py
import json
import random
from django import template
from django.utils.safestring import mark_safe

register = template.Library()

SERVER_URL = '{{.ServerURL}}'

@register.simple_tag(takes_context=True)
def headline_test(context, test_name, variants):
    """
    Usage: {% templatetag openblock %} headline_test "{{.TestName}}" variants {% templatetag closeblock %}
    Where variants is a list like ["Option A", "Option B"]
    """
    request = context['request']
    session = request.session

    key = f'ht_variant_{test_name}'
    if key not in session:
        session[key] = random.randint(0, len(variants) - 1)

    variant_index = session[key]
    text = variants[variant_index]
    visitor_id = getattr(request, 'ht_visitor_id', '')

    script = f'''
    <span data-ht-variant="{variant_index}">{text}</span>
    <script>
    fetch('{SERVER_URL}/b', {{
        method: 'POST',
        headers: {{ 'Content-Type': 'application/json' }},
        body: JSON.stringify({{
            t: '{test_name}',
            v: {variant_index},
            e: 'view',
            vid: '{visitor_id}'
        }})
    }}).catch(() => {{}});
    </script>
    '''
    return mark_safe(script)

@register.simple_tag(takes_context=True)
def convert_button(context, test_name, button_text):
    """
    Usage: {% templatetag openblock %} convert_button "{{.TestName}}" "Get Started" {% templatetag closeblock %}
    """
    request = context['request']
    session = request.session

    variant_index = session.get(f'ht_variant_{test_name}', 0)
    visitor_id = getattr(request, 'ht_visitor_id', '')

    script = f'''
    <button onclick="htConvert_{test_name}()">{button_text}</button>
    <script>
    function htConvert_{test_name}() {{
        fetch('{SERVER_URL}/b', {{
            method: 'POST',
            headers: {{ 'Content-Type': 'application/json' }},
            body: JSON.stringify({{
                t: '{test_name}',
                v: {variant_index},
                e: 'convert',
                vid: '{visitor_id}'
            }})
        }}).catch(() => {{}});
    }}
    </script>
    '''
    return mark_safe(script)
`
	rendered, _ := renderTemplate("templateTags", templateTags, data)
	files = append(files, SnippetFile{Filename: "headline_test.py", Content: rendered})

	return files, nil
}

// AllFrameworks returns all supported frameworks
func AllFrameworks() []Framework {
	return []Framework{
		FrameworkHTML,
		FrameworkNextJS,
		FrameworkReact,
		FrameworkVue,
		FrameworkSvelte,
		FrameworkLaravel,
		FrameworkDjango,
	}
}

// AllAnimations returns all supported animations
func AllAnimations() []Animation {
	return []Animation{
		AnimationScramble,
		AnimationPixel,
		AnimationTypewriter,
		AnimationNone,
	}
}
