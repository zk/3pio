class Threepio < Formula
  desc "Context-competent test runner for coding agents"
  homepage "https://github.com/zk/3pio"
  version "0.0.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/zk/3pio/releases/download/v#{version}/3pio-darwin-arm64.tar.gz"
      sha256 "" # This will be automatically updated by goreleaser
    else
      url "https://github.com/zk/3pio/releases/download/v#{version}/3pio-darwin-amd64.tar.gz" 
      sha256 "" # This will be automatically updated by goreleaser
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/zk/3pio/releases/download/v#{version}/3pio-linux-arm64.tar.gz"
      sha256 "" # This will be automatically updated by goreleaser
    else
      url "https://github.com/zk/3pio/releases/download/v#{version}/3pio-linux-amd64.tar.gz"
      sha256 "" # This will be automatically updated by goreleaser
    end
  end

  def install
    bin.install "3pio"
  end

  test do
    # Test that the binary runs and shows help
    system "#{bin}/3pio", "--help"
    
    # Test that the version flag works
    system "#{bin}/3pio", "--version"
  end

  def caveats
    <<~EOS
      3pio is a test runner adapter that works with Jest, Vitest, and pytest.
      
      Usage examples:
        3pio npx jest          # Run Jest tests
        3pio npx vitest run    # Run Vitest tests  
        3pio pytest           # Run pytest tests
        3pio npm test          # Run npm test script

      Reports are generated in .3pio/runs/ directory.
      
      For more information: https://github.com/zk/3pio
    EOS
  end
end