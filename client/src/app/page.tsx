import Link from "next/link";

export default function Home() {
  return (
    <div className="m-5 flex flex-col gap-3">
      <h1 className="text-xl font-medium">Orchestrator</h1>

      <div className="flex gap-3">
        <Link
          href="/login"
          className="px-2 py-1.5 text-sm bg-sky-600 text-white text-center rounded-md hover:bg-sky-700 transition-colors"
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

      <div className="text-sm text-neutral-400">
        &copy; {new Date().getFullYear()} Orchestrator
      </div>
    </div>
  );
}
