"use client"

import * as React from "react"
import { Clock } from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"

interface TimePickerProps {
  value?: string // HH:MM format
  onChange?: (time: string) => void
  placeholder?: string
  className?: string
  disabled?: boolean
}

export function TimePicker({
  value,
  onChange,
  placeholder = "Select time",
  className,
  disabled = false,
}: TimePickerProps) {
  const [open, setOpen] = React.useState(false)

  // Generate hours (00-23)
  const hours = Array.from({ length: 24 }, (_, i) => i.toString().padStart(2, "0"))
  // Generate minutes (00, 15, 30, 45)
  const minutes = ["00", "15", "30", "45"]

  const [selectedHour, selectedMinute] = value ? value.split(":") : ["", ""]

  const handleHourSelect = (hour: string) => {
    const newTime = `${hour}:${selectedMinute || "00"}`
    onChange?.(newTime)
  }

  const handleMinuteSelect = (minute: string) => {
    const newTime = `${selectedHour || "00"}:${minute}`
    onChange?.(newTime)
    setOpen(false)
  }

  const formatDisplayTime = (time: string) => {
    if (!time) return null
    const [h, m] = time.split(":")
    const hour = parseInt(h, 10)
    const ampm = hour >= 12 ? "PM" : "AM"
    const displayHour = hour % 12 || 12
    return `${displayHour}:${m} ${ampm}`
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          disabled={disabled}
          className={cn(
            "w-full justify-start text-left font-normal bg-input border-input",
            !value && "text-muted-foreground",
            className
          )}
        >
          <Clock className="mr-2 h-4 w-4" />
          {value ? formatDisplayTime(value) : <span>{placeholder}</span>}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <div className="flex">
          {/* Hours */}
          <div className="border-r">
            <div className="px-3 py-2 text-sm font-medium text-muted-foreground border-b">
              Hour
            </div>
            <div className="h-[200px] overflow-y-auto">
              <div className="flex flex-col p-1">
                {hours.map((hour) => (
                  <Button
                    key={hour}
                    variant={selectedHour === hour ? "default" : "ghost"}
                    size="sm"
                    className="justify-center"
                    onClick={() => handleHourSelect(hour)}
                  >
                    {hour}
                  </Button>
                ))}
              </div>
            </div>
          </div>
          {/* Minutes */}
          <div>
            <div className="px-3 py-2 text-sm font-medium text-muted-foreground border-b">
              Min
            </div>
            <div className="h-[200px] overflow-y-auto">
              <div className="flex flex-col p-1">
                {minutes.map((minute) => (
                  <Button
                    key={minute}
                    variant={selectedMinute === minute ? "default" : "ghost"}
                    size="sm"
                    className="justify-center"
                    onClick={() => handleMinuteSelect(minute)}
                  >
                    {minute}
                  </Button>
                ))}
              </div>
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
