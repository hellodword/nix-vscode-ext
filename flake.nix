{
  description = "some vscode extensions";

  outputs = { self, ... }:
    {
      vscode-extensions = builtins.fromJSON (builtins.readFile ./ext.json);
    };
}
