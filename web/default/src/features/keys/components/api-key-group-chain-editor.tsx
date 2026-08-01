/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  MouseSensor,
  TouchSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical, Trash2 } from 'lucide-react'
import { useMemo, type KeyboardEvent as ReactKeyboardEvent } from 'react'
import { useTranslation } from 'react-i18next'

import { Button, buttonVariants } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import {
  ApiKeyGroupMultiSelect,
  GroupRatioBadge,
  type ApiKeyGroupOption,
} from './api-key-group-combobox'

type ApiKeyGroupChainEditorProps = {
  options: ApiKeyGroupOption[]
  value: string[]
  onValueChange: (value: string[]) => void
}

type SortableGroupProps = {
  group: string
  index: number
  option?: ApiKeyGroupOption
  canRemove: boolean
  onRemove: () => void
}

function SortableGroup(props: SortableGroupProps) {
  const { t } = useTranslation()
  const sortable = useSortable({ id: props.group })
  const style = {
    transform: CSS.Transform.toString(sortable.transform),
    transition: sortable.transition,
  }
  const forwardSortableArrowKey = (
    event: ReactKeyboardEvent<HTMLButtonElement>
  ) => {
    if (
      !sortable.isDragging ||
      !['ArrowDown', 'ArrowLeft', 'ArrowRight', 'ArrowUp'].includes(event.code)
    ) {
      return
    }

    event.preventDefault()
    document.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: event.key,
        code: event.code,
        bubbles: true,
        cancelable: true,
      })
    )
  }

  return (
    <div
      ref={sortable.setNodeRef}
      style={style}
      className={cn(
        'bg-background flex min-h-14 w-full min-w-0 items-center gap-3 px-3 py-2',
        sortable.isDragging && 'relative z-10 shadow-lg'
      )}
    >
      <button
        ref={sortable.setActivatorNodeRef}
        type='button'
        className={cn(
          buttonVariants({ variant: 'ghost', size: 'icon-sm' }),
          'cursor-grab touch-none active:cursor-grabbing'
        )}
        aria-label={t('Drag to reorder')}
        title={t('Drag to reorder')}
        onKeyDownCapture={forwardSortableArrowKey}
        {...sortable.attributes}
        {...sortable.listeners}
      >
        <GripVertical />
      </button>
      <span className='bg-muted text-muted-foreground flex size-6 shrink-0 items-center justify-center rounded-md text-xs font-medium'>
        {props.index + 1}
      </span>
      <span className='min-w-0 flex-1'>
        <span className='block truncate text-sm font-medium'>
          {props.option?.label || props.group}
        </span>
        {props.option?.desc && (
          <span className='text-muted-foreground block truncate text-xs'>
            {props.option.desc}
          </span>
        )}
      </span>
      <GroupRatioBadge ratio={props.option?.ratio} />
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type='button'
              variant='ghost'
              size='icon-sm'
              disabled={!props.canRemove}
              onClick={props.onRemove}
              aria-label={t('Remove group')}
            />
          }
        >
          <Trash2 />
        </TooltipTrigger>
        <TooltipContent>{t('Remove group')}</TooltipContent>
      </Tooltip>
    </div>
  )
}

export function ApiKeyGroupChainEditor(props: ApiKeyGroupChainEditorProps) {
  const { t } = useTranslation()
  const optionMap = useMemo(
    () => new Map(props.options.map((option) => [option.value, option])),
    [props.options]
  )
  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 6 } }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 150, tolerance: 5 },
    }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  const handleDragEnd = (event: DragEndEvent) => {
    if (!event.over || event.active.id === event.over.id) return
    const oldIndex = props.value.indexOf(String(event.active.id))
    const newIndex = props.value.indexOf(String(event.over.id))
    if (oldIndex < 0 || newIndex < 0) return
    props.onValueChange(arrayMove(props.value, oldIndex, newIndex))
  }

  return (
    <div className='min-w-0 space-y-3'>
      <ApiKeyGroupMultiSelect
        options={props.options}
        value={props.value}
        onValueChange={props.onValueChange}
        placeholder={t('Select groups')}
        disabled={props.options.length === 0}
      />

      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
      >
        <SortableContext
          items={props.value}
          strategy={verticalListSortingStrategy}
        >
          <div className='w-full min-w-0 divide-y overflow-hidden rounded-lg border'>
            {props.value.map((group, index) => (
              <SortableGroup
                key={group}
                group={group}
                index={index}
                option={optionMap.get(group)}
                canRemove={props.value.length > 1}
                onRemove={() =>
                  props.onValueChange(
                    props.value.filter((value) => value !== group)
                  )
                }
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>
    </div>
  )
}
