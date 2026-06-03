import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

// cn — merge Tailwind classes safely (shadcn/ui convention)
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
