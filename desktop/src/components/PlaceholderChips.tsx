import { Group, Badge } from '@mantine/core'
import { useT } from '../i18n'

const placeholders = [
  { key: 'number', value: '((number))' },
  { key: 'customer', value: '((customer))' },
  { key: 'total', value: '((total))' },
  { key: 'due_date', value: '((due_date))' },
  { key: 'issue_date', value: '((issue_date))' },
  { key: 'supplier', value: '((supplier))' },
]

// Translation keys for placeholder labels
const placeholderLabels: Record<string, string> = {
  number: 'email.ph_number',
  customer: 'email.ph_customer',
  total: 'email.ph_total',
  due_date: 'email.ph_due_date',
  issue_date: 'email.ph_issue_date',
  supplier: 'email.ph_supplier',
}

interface Props {
  onInsert: (placeholder: string) => void
}

export function PlaceholderChips({ onInsert }: Props) {
  const { t } = useT()
  return (
    <Group gap={4}>
      {placeholders.map((p) => (
        <Badge
          key={p.value}
          variant="light"
          color="blue"
          size="sm"
          style={{ cursor: 'pointer' }}
          onClick={() => onInsert(p.value)}
        >
          {t(placeholderLabels[p.key])}
        </Badge>
      ))}
    </Group>
  )
}
