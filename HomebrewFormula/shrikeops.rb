class Shrikeops < Formula
  desc "Kubernetes security and compliance scanner CLI"
  homepage "https://github.com/warbletech/shrikeops"
  version "0.1.0"
  license "Apache-2.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/warbletech/shrikeops/releases/download/v#{version}/shrikeops-darwin-arm64"
      sha256 "" # TODO: Add SHA256 after first release
    else
      url "https://github.com/warbletech/shrikeops/releases/download/v#{version}/shrikeops-darwin-amd64"
      sha256 "" # TODO: Add SHA256 after first release
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/warbletech/shrikeops/releases/download/v#{version}/shrikeops-linux-arm64"
      sha256 "" # TODO: Add SHA256 after first release
    else
      url "https://github.com/warbletech/shrikeops/releases/download/v#{version}/shrikeops-linux-amd64"
      sha256 "" # TODO: Add SHA256 after first release
    end
  end

  def install
    bin.install Dir["shrikeops-*"].first => "shrikeops"
  end

  test do
    system "#{bin}/shrikeops", "--version"
  end
end
