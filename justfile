nyaa:
	sudo nixos-rebuild switch --flake .#nyaa
wawa:
	sudo nixos-rebuild switch --flake .#wawa
waybar:
	pkill waybar && hyprctl dispatch exec waybar
