# Homebrew Tap Setup for ShrikeOps

## Overview

Homebrew taps allow macOS and Linux users to install ShrikeOps CLI with:
```bash
brew tap warbletech/tap
brew install shrikeops
```

## Option 1: Separate Tap Repository (Recommended)

### Step 1: Create Homebrew Tap Repo

Create a new GitHub repository: `warbletech/homebrew-tap`

Repository structure:
```
homebrew-tap/
├── Formula/
│   └── shrikeops.rb          # Main formula
├── README.md
└── .github/
    └── workflows/
        └── update-formula.yml  # Auto-update on release
```

### Step 2: Create the Formula

File: `Formula/shrikeops.rb`

```ruby
class Shrikeops < Formula
  desc "Kubernetes security and compliance scanner CLI"
  homepage "https://github.com/warbletech/shrikeops"
  version "0.1.0"
  license "Apache-2.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/warbletech/shrikeops/releases/download/v#{version}/shrikeops-darwin-arm64"
      sha256 "PLACEHOLDER_SHA256"
    else
      url "https://github.com/warbletech/shrikeops/releases/download/v#{version}/shrikeops-darwin-amd64"
      sha256 "PLACEHOLDER_SHA256"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/warbletech/shrikeops/releases/download/v#{version}/shrikeops-linux-arm64"
      sha256 "PLACEHOLDER_SHA256"
    else
      url "https://github.com/warbletech/shrikeops/releases/download/v#{version}/shrikeops-linux-amd64"
      sha256 "PLACEHOLDER_SHA256"
    end
  end

  def install
    bin.install Dir["shrikeops-*"].first => "shrikeops"
  end

  test do
    system "#{bin}/shrikeops", "--version"
  end
end
```

### Step 3: Auto-Update Formula on Release

Create `.github/workflows/update-formula.yml` in the tap repo:

```yaml
name: Update Formula on Release

on:
  repository_dispatch:
    types: [release-published]
  workflow_dispatch:
    inputs:
      version:
        required: true
        type: string

jobs:
  update-formula:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Download binaries and compute SHA256
        id: sha256
        run: |
          VERSION="${{ github.event.client_payload.version || inputs.version }}"
          for platform in darwin-amd64 darwin-arm64 linux-amd64 linux-arm64; do
            curl -sL "https://github.com/warbletech/shrikeops/releases/download/${VERSION}/shrikeops-${platform}" -o /tmp/shrikeops-${platform}
            echo "${platform}_sha256=$(sha256sum /tmp/shrikeops-${platform} | cut -d' ' -f1)" >> $GITHUB_OUTPUT
          done

      - name: Update formula
        run: |
          cat > Formula/shrikeops.rb << 'EOF'
          class Shrikeops < Formula
            desc "Kubernetes security and compliance scanner CLI"
            homepage "https://github.com/warbletech/shrikeops"
            version "${{ github.event.client_payload.version || inputs.version }}"
            license "Apache-2.0"

            on_macos do
              if Hardware::CPU.arm?
                url "https://github.com/warbletech/shrikeops/releases/download/v#{version}/shrikeops-darwin-arm64"
                sha256 "${{ steps.sha256.outputs.darwin-arm64_sha256 }}"
              else
                url "https://github.com/warbletech/shrikeops/releases/download/v#{version}/shrikeops-darwin-amd64"
                sha256 "${{ steps.sha256.outputs.darwin-amd64_sha256 }}"
              end
            end

            on_linux do
              if Hardware::CPU.arm?
                url "https://github.com/warbletech/shrikeops/releases/download/v#{version}/shrikeops-linux-arm64"
                sha256 "${{ steps.sha256.outputs.linux-arm64_sha256 }}"
              else
                url "https://github.com/warbletech/shrikeops/releases/download/v#{version}/shrikeops-linux-amd64"
                sha256 "${{ steps.sha256.outputs.linux-amd64_sha256 }}"
              end
            end

            def install
              bin.install Dir["shrikeops-*"].first => "shrikeops"
            end

            test do
              system "#{bin}/shrikeops", "--version"
            end
          end
          EOF

      - name: Commit and push
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add Formula/shrikeops.rb
          git commit -m "Update shrikeops to ${{ github.event.client_payload.version || inputs.version }}"
          git push
```

### Step 4: Trigger Formula Update from ShrikeOps

Add this step to the end of `publish.yml` in the ShrikeOps repo:

```yaml
      - name: Trigger tap formula update
        uses: peter-evans/repository-dispatch@v3
        with:
          token: ${{ secrets.TAP_REPO_TOKEN }}
          repository: warbletech/homebrew-tap
          event-type: release-published
          client-payload: '{"version": "${{ github.ref_name }}"}'
```

Create a Personal Access Token (PAT) with `repo` scope and add it as `TAP_REPO_TOKEN` secret.

### Step 5: README for Tap Repo

```markdown
# WarbleTech Homebrew Tap

## Quick Start

```bash
# Add the tap
brew tap warbletech/tap

# Install ShrikeOps
brew install shrikeops

# Upgrade
brew upgrade shrikeops
```

## Available Formulae

- `shrikeops` — Kubernetes security scanner
```

## Option 2: Built-in Formula (Current Repo)

Users can install directly without a tap:

```bash
brew install --HEAD https://raw.githubusercontent.com/warbletech/shrikeops/main/HomebrewFormula/shrikeops.rb
```

Or with specific version:
```bash
brew install https://github.com/warbletech/shrikeops/releases/download/v0.1.0/shrikeops.rb
```

## Testing the Formula Locally

```bash
# Download formula
curl -O https://raw.githubusercontent.com/warbletech/homebrew-tap/main/Formula/shrikeops.rb

# Install from local file
brew install --formula ./shrikeops.rb

# Test
shrikeops --version
```

## SHA256 Calculation

After binaries are published, get SHA256 hashes:

```bash
# macOS (Intel)
curl -sL https://github.com/warbletech/shrikeops/releases/download/v0.1.0/shrikeops-darwin-amd64 | shasum -a 256

# macOS (Apple Silicon)
curl -sL https://github.com/warbletech/shrikeops/releases/download/v0.1.0/shrikeops-darwin-arm64 | shasum -a 256

# Linux (AMD64)
curl -sL https://github.com/warbletech/shrikeops/releases/download/v0.1.0/shrikeops-linux-amd64 | shasum -a 256

# Linux (ARM64)
curl -sL https://github.com/warbletech/shrikeops/releases/download/v0.1.0/shrikeops-linux-arm64 | shasum -a 256
```
