{
  description = "存放不是秘密但不想公开的数据";

  inputs = {
    # https://nixos.org/manual/nixos/unstable/options.html
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable"; # "github:NixOS/nixpkgs/release-22.11";
  };

  outputs = { self, nixpkgs, ... }:
    let
      # https://github.com/badly-drawn-wizards/dotfiles/blob/0ec000db9be631d1d339a195c76d308612df2a52/home-manager/editors/vscode/default.nix#L3-L11
      inherit (builtins) head isString concatLists concatStringsSep fromJSON map;
      inherit (nixpkgs.lib.strings) splitString;
      fromJSONC = jsonc:
        let
          linesWithSep = concatLists (map (l: if isString l then [ l ] else l) (builtins.split "([\r\n]+)" jsonc));
          removeComment = line: head (splitString "//" line);
          json = concatStringsSep "" (map removeComment linesWithSep);
        in
        fromJSON json;
    in
    {
      vscode-extensions = fromJSONC (builtins.readFile ./ext.jsonc);
    };
}
