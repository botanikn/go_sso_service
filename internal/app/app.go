package app

import (
	"log/slog"
	"time"

	"github.com/botanikn/go_sso_service/internal/app/grpcapp"
	"github.com/botanikn/go_sso_service/internal/config"
	"github.com/botanikn/go_sso_service/pkg/database"
)

// COMMENT может быть приватной, тоже не крит в этом месте, но в го и так плохо все с защитой данных,
// в целом советую вообще отказаться от публичных структур для сервисов, они не нужны почти никогда,
// это заставит тебя исполььзовать интерфейсы, потом проще будет с тестированием
type App struct {
	GRPCSrv *grpcapp.App // COMMENT  поле тоже нет необходимости делать побличным, если прям очень хотелось инициализацию app разбить на 2 этапа, то есть встраивание
	// application.GRPCSrv.MustRun() в main смотрится странно, я хочу просто запустить app, мне не интересно что у него внутри,
	//  ну условно если у тебя еще появляется http сервер ты не должен менять main, в твоем коде за запуск отвечает app
}

func New(
	log *slog.Logger,
	grpcPort int,
	storageCfg *config.DbConfig,
	tokenTTL time.Duration,
) *App {
	// COMMENT а дб почему тогда не отдельный сервис?
	DB, err := database.NewDB(storageCfg)

	if err != nil {
		panic("failed to connect to the database: " + err.Error())
	}

	grpcApp := grpcapp.New(log, grpcPort, DB, tokenTTL)

	return &App{
		GRPCSrv: grpcApp,
	}
}

// те дописываем
// func(a *App) MustRun(ctx context.Context) {
// 	a.GRPCSrv.MustRun(ctx)
// 	остальные сервисы когда будут
// }
