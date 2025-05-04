"use client";

import { useUser } from "@/context/UserContext";
import { useRouter } from "next/navigation";
import { TbUser, TbSettings, TbLogout } from "react-icons/tb";
import Link from "next/link";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  DropdownMenuItemDesctructive,
} from "@/components/ui/dropdown-menu";
import { GoTriangleDown } from "react-icons/go";

export default function Navbar() {
  const { user, logout } = useUser();
  const router = useRouter();

  const handleLogout = () => {
    logout();
    router.push("/");
  };

  return (
    <header className="border-b border-neutral-200">
      <div className="flex items-center justify-between p-4">
        <Link href={user ? "/home" : "/"} className="text-lg font-medium">
          Orchestrator
        </Link>

        {user ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button className="cursor-pointer flex items-center text-sm gap-2">
                Account
                <GoTriangleDown size="20px" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent className="w-48">
              <DropdownMenuLabel>Welcome, {user.firstName}!</DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuItem className="flex item-center gap-1.5 group">
                  <TbUser className="text-neutral-700 group-hover:text-white" />
                  Profile
                </DropdownMenuItem>
                <DropdownMenuItem className="flex item-center gap-1.5 group">
                  <TbSettings className="text-neutral-700 group-hover:text-white" />
                  Settings
                </DropdownMenuItem>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuItemDesctructive
                className="flex items-center gap-1.5 group"
                onClick={handleLogout}
              >
                <TbLogout className="text-neutral-700 group-hover:text-white" />
                Log out
              </DropdownMenuItemDesctructive>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : (
          <div className="flex gap-3">
            <Link
              href="/login"
              className="px-2 py-1 text-sm bg-sky-600 text-white text-center rounded-md hover:bg-sky-700 transition-colors"
            >
              Login
            </Link>
            <Link
              href="/register"
              className="px-2 py-1 text-sm border border-neutral-300 text-neutral-600 text-center rounded-md hover:bg-neutral-100 transition-colors"
            >
              Register
            </Link>
          </div>
        )}
      </div>
    </header>
  );
}
