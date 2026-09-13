# typed: false
# frozen_string_literal: true

# Tap formula (not homebrew-core). URLs and sha256 values are placeholders
# until a maintainer pushes a v* tag; GoReleaser then overwrites this file
# with the GitHub Release assets and checksums for that version.
class Killport < Formula
  desc "Free a TCP Port by terminating its Occupant"
  homepage "https://github.com/jawharmed/kill-port"
  version "0.0.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/jawharmed/kill-port/releases/download/v0.0.0/killport_Darwin_x86_64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
    on_arm do
      url "https://github.com/jawharmed/kill-port/releases/download/v0.0.0/killport_Darwin_arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/jawharmed/kill-port/releases/download/v0.0.0/killport_Linux_x86_64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  def install
    bin.install "killport"
  end

  test do
    assert_match "killport version", shell_output("#{bin}/killport --version")
  end
end
