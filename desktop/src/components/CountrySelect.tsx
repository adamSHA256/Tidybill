import { Select, Modal, TextInput, Button, Group, Stack } from '@mantine/core'
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { useT } from '../i18n'

const BASE_COUNTRIES = ['CZ', 'SK', 'DE', 'AT', 'PL', 'HU']
const ADD_COUNTRY = '__add_country__'

interface CountrySelectProps {
  value: string
  onChange: (value: string) => void
  label?: string
  searchable?: boolean
}

export function CountrySelect({ value, onChange, label, searchable }: CountrySelectProps) {
  const { t } = useT()
  const queryClient = useQueryClient()
  const [modalOpen, setModalOpen] = useState(false)
  const [newCode, setNewCode] = useState('')

  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: api.getSettings,
  })

  const updateSettings = useMutation({
    mutationFn: api.updateSettings,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['settings'] }),
  })

  const customCountries: string[] = (() => {
    try { return JSON.parse(settings?.custom_countries || '[]') } catch { return [] }
  })()

  const allCountries = [...new Set([...BASE_COUNTRIES, ...customCountries])]
  const data = [
    ...allCountries.map((c) => ({ value: c, label: c })),
    { value: ADD_COUNTRY, label: `+ ${t('country.add')}` },
  ]

  const handleChange = (v: string | null) => {
    if (v === ADD_COUNTRY) {
      setNewCode('')
      setModalOpen(true)
      return
    }
    if (v) onChange(v)
  }

  const handleAdd = () => {
    const code = newCode.trim().toUpperCase()
    if (!code) return
    const updated = [...new Set([...customCountries, code])]
    updateSettings.mutate({ custom_countries: JSON.stringify(updated) })
    onChange(code)
    setModalOpen(false)
  }

  return (
    <>
      <Select
        label={label ?? t('country.label')}
        data={data}
        value={value}
        onChange={handleChange}
        searchable={searchable}
      />
      <Modal opened={modalOpen} onClose={() => setModalOpen(false)}
        title={t('country.add')} size="xs">
        <Stack gap="md">
          <TextInput label={t('country.code')} placeholder="US"
            value={newCode} onChange={(e) => setNewCode(e.currentTarget.value.toUpperCase())}
            maxLength={5}
            onKeyDown={(e) => { if (e.key === 'Enter') handleAdd() }}
          />
          <Group justify="end">
            <Button variant="default" onClick={() => setModalOpen(false)}>{t('common.cancel')}</Button>
            <Button onClick={handleAdd} disabled={!newCode.trim()}>{t('common.save')}</Button>
          </Group>
        </Stack>
      </Modal>
    </>
  )
}
