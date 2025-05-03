

"use client";

import { useEffect } from "react";
import { useUser } from "@/context/UserContext";
import { useRouter } from "next/navigation";
import { TbUser, TbSettings, TbLogout } from "react-icons/tb";

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
import Link from "next/link";


export default function Account() {
  const { user, loading, logout } = useUser();
  const router = useRouter();

  useEffect(() => {
    // If user is not logged in and not currently loading, redirect to login
    if (!loading && !user) {
      router.push("/login");
    }
  }, [user, loading, router]);

  // Show loading state while checking authentication
  if (loading) {
    return (
      <div className="m-5">
        <p className="text-sm text-neutral-500">Loading...</p>
      </div>
    );
  }

  // Display nothing if user is logged in, this will be briefly shown before the redirect happens
  if (!user) {
    return null;
  }

  const handleLogout = () => {
    logout();
    router.push("/");
  };

  return (
    <div className="flex flex-col">
      {/* Navbar with dropdown */}
      <header className="border-b border-neutral-200">
        <div className="flex items-center justify-between p-4">
          <Link href="/" className="text-lg font-medium">Orchestrator</Link>
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
                <DropdownMenuItem className="flex item-center gap-1.5 group"><TbUser className="text-neutral-700 group-hover:text-white" />Profile</DropdownMenuItem>
                <DropdownMenuItem className="flex item-center gap-1.5 group"><TbSettings className="text-neutral-700 group-hover:text-white" />Settings</DropdownMenuItem>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuItemDesctructive className="flex items-center gap-1.5 group" onClick={handleLogout}><TbLogout className="text-neutral-700 group-hover:text-white" />Log out</DropdownMenuItemDesctructive>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>

      {/* Main content */}
      <div className="m-5 flex flex-col gap-4">
        <h1 className="text-lg font-medium">Profile</h1>

        <div className="flex flex-col gap-3">

            <div className="flex flex-col gap-2">
              <div className="flex">
                <span className="w-20 text-sm text-neutral-500">Name:</span>
                <span className="text-sm">
                  {user.firstName} {user.lastName}
                </span>
              </div>
              <div className="flex">
                <span className="w-20 text-sm text-neutral-500">Email:</span>
                <span className="text-sm">{user.email}</span>
              </div>
              <div className="flex">
                <span className="w-20 text-sm text-neutral-500">User ID:</span>
                <span className="text-sm text-neutral-500">{user.id}</span>
              </div>
          </div>
        </div>
      </div>
    </div>
  );
}       
