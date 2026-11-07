// Global reusable Tabs primitive built on @base-ui/react/tabs following Shadcn UI conventions.
// Provides accessible keyboard navigation and panel switching without custom color utilities.
"use client"

import * as React from "react"
import { Tabs as TabsPrimitive } from "@base-ui/react/tabs"
import { cn } from "cn"

// Tabs manages the active tab state and coordinates TabsList and TabsContent children.
function Tabs({ className, ...props }: TabsPrimitive.Root.Props) {
  return (
    <TabsPrimitive.Root
      data-slot="tabs"
      className={cn("flex flex-col gap-4", className)}
      {...props}
    />
  )
}

// TabsList renders the accessible tablist container holding TabsTrigger buttons.
function TabsList({ className, ...props }: TabsPrimitive.List.Props) {
  return (
    <TabsPrimitive.List
      data-slot="tabs-list"
      className={cn(
        "inline-flex h-10 w-full items-center justify-center rounded-lg border p-1",
        className
      )}
      {...props}
    />
  )
}

// TabsTrigger renders an individual tab button that activates its corresponding TabsContent panel.
function TabsTrigger({ className, ...props }: TabsPrimitive.Tab.Props) {
  return (
    <TabsPrimitive.Tab
      data-slot="tabs-trigger"
      className={cn(
        "inline-flex flex-1 items-center justify-center rounded-md px-3 py-1.5 text-sm font-medium whitespace-nowrap transition-all outline-none select-none disabled:pointer-events-none disabled:opacity-50 data-active:border data-active:font-semibold data-active:shadow-xs",
        className
      )}
      {...props}
    />
  )
}

// TabsContent renders the panel content associated with a matching TabsTrigger value.
function TabsContent({ className, ...props }: TabsPrimitive.Panel.Props) {
  return (
    <TabsPrimitive.Panel
      data-slot="tabs-content"
      className={cn("flex-1 outline-none", className)}
      {...props}
    />
  )
}

export { Tabs, TabsList, TabsTrigger, TabsContent }
