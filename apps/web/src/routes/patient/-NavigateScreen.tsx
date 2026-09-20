import { LanguageToggle, useT } from '@/i18n'
import {
  AppBar,
  Divider,
  FloorPlanMap,
  type FloorPlanRoute,
  InfoNote,
  LinkButton,
  Screen,
  ScreenDock,
} from '@/design-system'
import { floorPlanFor, type DestinationPlan, type NavigatePlan } from '@/features/floorplan'

/**
 * Patient · navigate to a service point — the real floor-plan asset with the
 * destination room highlighted and named (#25), plus, when the current
 * location is known, the walking line and turn-by-turn cues from the
 * navigation API (#28) drawn as a live overlay (#29). Without a location
 * the screen keeps its honest destination-only view — nothing about the
 * route is invented (DESIGN.md). The plan always comes from the caller:
 * the live route computes it from the journey, /design passes its static
 * reference plan.
 */
export function NavigateScreen({
  plan,
  route,
  currentLocation,
  onBack,
}: {
  plan: NavigatePlan
  route?: { mapRoute: FloorPlanRoute; cues: string[] }
  /** Plain line stating where the patient is (#35); absent when unknown. */
  currentLocation?: string
  onBack?: () => void
}) {
  const t = useT()

  return (
    <Screen variant="patient">
      <AppBar
        onBack={onBack}
        backLabel={t('navigate.back')}
        title={appBarTitle(plan, t)}
        subtitle={plan.state === 'plan' ? plan.subtitle : undefined}
        trailing={<LanguageToggle />}
      />

      {plan.state === 'plan' && floorPlanFor(plan.floorId) ? (
        <>
          <div className="flex-1 overflow-hidden px-gutter pt-1 pb-2">
            <FloorPlanMap
              svg={floorPlanFor(plan.floorId) ?? ''}
              floorLabel={plan.floorLabel}
              destination={{
                placeId: plan.placeId,
                name: plan.name,
                x: plan.x,
                y: plan.y,
              }}
              route={route?.mapRoute}
              ariaLabel={
                route
                  ? t('map.routeAria', { floor: plan.floorLabel, name: plan.name })
                  : t('map.planAria', { floor: plan.floorLabel, name: plan.name })
              }
              labels={{
                viewFullFloor: t('map.viewFullFloor'),
                viewRoute: t('map.viewRoute'),
                viewDestination: t('map.viewDestination'),
                youAreHere: t('map.youAreHere'),
              }}
            />
          </div>

          <ScreenDock>
            <DestinationPanel
              plan={plan}
              cues={route?.cues}
              currentLocation={currentLocation}
            />
          </ScreenDock>
        </>
      ) : (
        <div className="px-gutter pt-5">
          <InfoNote>{noticeFor(plan, t)}</InfoNote>
        </div>
      )}
    </Screen>
  )
}

type Translate = ReturnType<typeof useT>

function appBarTitle(plan: NavigatePlan, t: Translate): string {
  switch (plan.state) {
    case 'pending':
      return t('navigate.loadingTitle')
    case 'no-destination':
      return t('navigate.noDestinationTitle')
    default:
      return plan.title
  }
}

function noticeFor(plan: NavigatePlan, t: Translate): string {
  switch (plan.state) {
    case 'pending':
      return t('navigate.pendingNotice')
    case 'no-destination':
      return t('navigate.noDestinationNotice')
    default:
      return t('navigate.unsupportedNotice')
  }
}

/**
 * The sheet under the map carries the destination facts patients need —
 * name, floor, place — and, once the current location is known, the
 * turn-by-turn cues derived from the route. Without a location it says so
 * instead of guessing a route (DESIGN.md's honesty rule).
 */
function DestinationPanel({
  plan,
  cues,
  currentLocation,
}: {
  plan: DestinationPlan
  cues?: string[]
  currentLocation?: string
}) {
  const t = useT()

  return (
    <div className="flex flex-col gap-3.5 rounded-t-xl bg-surface px-gutter pt-3 pb-6 shadow-sheet">
      <div className="mx-auto h-1 w-9 rounded-full bg-line" aria-hidden="true" />
      <div className="flex items-baseline justify-between gap-3">
        <div className="min-w-0">
          <div className="font-sans text-body-md font-bold text-ink">{plan.name}</div>
          <div className="font-sans text-body-sm text-ink-muted">{plan.subtitle}</div>
        </div>
        <span className="shrink-0 rounded-full border border-line bg-neutral px-2.5 py-1 font-sans text-caption font-bold text-ink-muted">
          {plan.floorLabel}
        </span>
      </div>
      <Divider />
      {/* Where the patient stands (#35) — stated once, plainly; the fallback
          below says so when it is not yet known. */}
      {currentLocation && (
        <p className="m-0 font-sans text-caption font-bold text-secondary">{currentLocation}</p>
      )}
      {cues ? (
        <ol className="m-0 flex list-decimal flex-col gap-1.5 pl-5">
          {cues.map((cue) => (
            <li key={cue} className="font-sans text-body-sm text-ink">
              {cue}
            </li>
          ))}
        </ol>
      ) : (
        <p className="m-0 font-sans text-caption text-ink-muted">{t('navigate.waitingLocation')}</p>
      )}
      <LinkButton>{t('navigate.askStaffIfLost')}</LinkButton>
    </div>
  )
}
