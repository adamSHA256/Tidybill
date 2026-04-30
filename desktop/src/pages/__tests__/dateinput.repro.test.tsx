import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { MantineProvider } from '@mantine/core'
import { DateInput, DatesProvider } from '@mantine/dates'
import dayjs from 'dayjs'
import customParseFormat from 'dayjs/plugin/customParseFormat'
import 'dayjs/locale/cs'

// Mirror main.tsx so the test reproduces real runtime behavior.
dayjs.extend(customParseFormat)

function Wrap() {
  const [val, setVal] = useState<string | null>('2026-04-15')
  return (
    <MantineProvider>
      <DatesProvider settings={{ locale: 'cs' }}>
        <div data-testid="dump">{String(val)}</div>
        <DateInput aria-label="d" valueFormat="DD.MM.YYYY" value={val} onChange={setVal} clearable />
      </DatesProvider>
    </MantineProvider>
  )
}

describe('DateInput repro — Czech format', () => {
  it('clicking a calendar day stores the YYYY-MM-DD of that day', async () => {
    const user = userEvent.setup()
    render(<Wrap />)
    expect(screen.getByTestId('dump').textContent).toBe('2026-04-15')
    const input = screen.getByLabelText('d') as HTMLInputElement
    expect(input.value).toBe('15.04.2026')

    // Open calendar by focusing the input
    await user.click(input)
    // Wait for the calendar to populate days
    await new Promise(resolve => setTimeout(resolve, 100))
    const allButtons = document.querySelectorAll('button')
    const dayButton = Array.from(allButtons).find(b => b.textContent?.trim() === '16') as HTMLElement | undefined
    if (!dayButton) throw new Error('No day "16" among ' + allButtons.length + ': ' + Array.from(allButtons).map(b => b.textContent).join('|'))
    await user.click(dayButton)

    expect(screen.getByTestId('dump').textContent).toBe('2026-04-16')
    expect(input.value).toBe('16.04.2026')
  })

  it('typing 03.02.2026 stores 2026-02-03', async () => {
    const user = userEvent.setup()
    render(<Wrap />)
    const input = screen.getByLabelText('d') as HTMLInputElement
    await user.click(input)
    await user.clear(input)
    await user.type(input, '03.02.2026')
    // Blur
    await user.tab()
    expect(screen.getByTestId('dump').textContent).toBe('2026-02-03')
  })
})
