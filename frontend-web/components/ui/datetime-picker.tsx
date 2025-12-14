"use client"

import * as React from "react"
import { format, parse } from "date-fns"
import { Calendar as CalendarIcon, Clock } from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"

interface DateTimePickerProps {
  value?: string // ISO string or datetime-local format (YYYY-MM-DDTHH:MM)
  onChange?: (value: string) => void
  placeholder?: string
  className?: string
  disabled?: boolean
}

export function DateTimePicker({
  value,
  onChange,
  placeholder = "Pick date and time",
  className,
  disabled = false,
}: DateTimePickerProps) {
  const [open, setOpen] = React.useState(false)

  // Parse value to Date and time string
  const parseValue = (val: string | undefined) => {
    if (!val) return { date: undefined, hour: "00", minute: "00" }
    try {
      const date = new Date(val)
      if (isNaN(date.getTime())) return { date: undefined, hour: "00", minute: "00" }
      return {
        date,
        hour: date.getHours().toString().padStart(2, "0"),
        minute: date.getMinutes().toString().padStart(2, "0"),
      }
    } catch {
      return { date: undefined, hour: "00", minute: "00" }
    }
  }

  const { date: selectedDate, hour: selectedHour, minute: selectedMinute } = parseValue(value)

  // Generate hours (00-23)
  const hours = Array.from({ length: 24 }, (_, i) => i.toString().padStart(2, "0"))
  // Generate minutes (00, 15, 30, 45)
  const minutes = ["00", "15", "30", "45"]

  const updateDateTime = (newDate?: Date, newHour?: string, newMinute?: string) => {
    const d = newDate || selectedDate || new Date()
    const h = newHour ?? selectedHour
    const m = newMinute ?? selectedMinute

    d.setHours(parseInt(h, 10))
    d.setMinutes(parseInt(m, 10))
    d.setSeconds(0)
    d.setMilliseconds(0)

    // Return in datetime-local format: YYYY-MM-DDTHH:MM
    const formatted = format(d, "yyyy-MM-dd'T'HH:mm")
    onChange?.(formatted)
  }

  const handleDateSelect = (date: Date | undefined) => {
    if (date) {
      updateDateTime(date)
    }
  }

  const handleHourSelect = (hour: string) => {
    updateDateTime(undefined, hour)
  }

  const handleMinuteSelect = (minute: string) => {
    updateDateTime(undefined, undefined, minute)
  }

  const formatDisplayValue = () => {
    if (!selectedDate) return null
    const h = parseInt(selectedHour, 10)
    const ampm = h >= 12 ? "PM" : "AM"
    const displayHour = h % 12 || 12
    return `${format(selectedDate, "PPP")} ${displayHour}:${selectedMinute} ${ampm}`
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
          <CalendarIcon className="mr-2 h-4 w-4" />
          {selectedDate ? formatDisplayValue() : <span>{placeholder}</span>}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <div className="flex">
          {/* Calendar */}
          <div className="border-r">
            <Calendar
              mode="single"
              selected={selectedDate}
              onSelect={handleDateSelect}
              initialFocus
            />
          </div>
          {/* Time Picker */}
          <div className="flex flex-col">
            <div className="flex items-center gap-1 px-3 py-2 border-b">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm font-medium">Time</span>
            </div>
            <div className="flex flex-1">
              {/* Hours */}
              <div className="border-r">
                <div className="px-2 py-1 text-xs text-muted-foreground text-center border-b">
                  Hr
                </div>
                <div className="h-[200px] overflow-y-auto">
                  <div className="flex flex-col p-1">
                    {hours.map((hour) => (
                      <Button
                        key={hour}
                        variant={selectedHour === hour ? "default" : "ghost"}
                        size="sm"
                        className="h-7 w-10 justify-center text-xs"
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
                <div className="px-2 py-1 text-xs text-muted-foreground text-center border-b">
                  Min
                </div>
                <div className="h-[200px] overflow-y-auto">
                  <div className="flex flex-col p-1">
                    {minutes.map((minute) => (
                      <Button
                        key={minute}
                        variant={selectedMinute === minute ? "default" : "ghost"}
                        size="sm"
                        className="h-7 w-10 justify-center text-xs"
                        onClick={() => handleMinuteSelect(minute)}
                      >
                        {minute}
                      </Button>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        {/* Footer with Done button */}
        <div className="border-t p-2 flex justify-end">
          <Button size="sm" onClick={() => setOpen(false)}>
            Done
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  )
}
