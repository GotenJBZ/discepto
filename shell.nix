{pkgs ? import <nixos-unstable> {}}:
pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    gopls
    delve
  ];
  hardeningDisable = [ "all" ];
}
