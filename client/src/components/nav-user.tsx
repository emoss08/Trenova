"use client";

import {
	Avatar,
	AvatarFallback,
	AvatarImage,
} from "@/components/ui/avatar";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuGroup,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { UserIcon, SettingsIcon, CreditCardIcon, LogOutIcon } from "lucide-react";

const user = {
	name: "Shaban Haider",
	email: "shaban@efferd.com",
	avatar: "https://github.com/shabanhr.png",
};

export function NavUser() {
	return (
		<DropdownMenu>
			<DropdownMenuTrigger
				render={
					<Avatar className="size-8">
						<AvatarImage src={user.avatar} />
						<AvatarFallback>{user.name.charAt(0)}</AvatarFallback>
					</Avatar>
				}
			/>
			<DropdownMenuContent align="end" className="w-60">
				<DropdownMenuItem
					className="flex items-center justify-start gap-2"
					title={user.name}
					description={user.email}
					startContent={
						<Avatar className="size-10">
							<AvatarImage src={user.avatar} />
							<AvatarFallback>{user.name.charAt(0)}</AvatarFallback>
						</Avatar>
					}
					titleClassProps="font-medium text-foreground"
					descriptionClassProps="max-w-full overflow-hidden overflow-ellipsis whitespace-nowrap text-muted-foreground text-xs"
				/>
				<DropdownMenuSeparator />
				<DropdownMenuGroup>
					<DropdownMenuItem title="Account" startContent={<UserIcon />} />
					<DropdownMenuItem title="Settings" startContent={<SettingsIcon />} />
				</DropdownMenuGroup>
				<DropdownMenuSeparator />
				<DropdownMenuGroup>
					<DropdownMenuItem
						title="Plan & Billing"
						startContent={<CreditCardIcon />}
					/>
				</DropdownMenuGroup>
				<DropdownMenuSeparator />
				<DropdownMenuGroup>
					<DropdownMenuItem
						className="w-full cursor-pointer"
						color="danger"
						title="Log out"
						startContent={<LogOutIcon />}
					/>
				</DropdownMenuGroup>
			</DropdownMenuContent>
		</DropdownMenu>
	);
}
